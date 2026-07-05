# Auto-fix Workflow (`--auto-fix`)

Vòng lặp patch tự động cho findings CRITICAL/HIGH sau khi report đã render xong.

> Được gọi từ [`../SKILL.md`](../SKILL.md) **Step 4c** khi user truyền flag `--auto-fix` — chạy TRƯỚC khi report cuối được render ở Step 5, để `patch_status` có mặt trong JSON summary và section Auto-fix. File này giả định `findings[]` (canonical, xem [`../references/output-format.md`](../references/output-format.md)) đã có từ Step 4 (+ Step 4b nếu có `--sca`).

**Đây là bước duy nhất trong skill được phép ghi đè file nguồn của user.** Mọi bước khác của vbsec chỉ đọc + báo cáo.

## Inputs

- `$AUTO_FIX` — `true` nếu flag được truyền (xem SKILL.md Step 0)
- `$IS_GIT_REPO` — bắt buộc `true` để chạy bước này
- `$SCAN_ROOT` — `.` khi quét working tree; thư mục snapshot tạm khi scope là `commit id` / `pr id` (xem SKILL.md Step 0)
- `$PRIMARY_LANG` — dùng để chọn verify command
- `findings[]` đã render ở Step 5 (mỗi finding có `file`, `line`, `rule_id`, `severity`)

## Gate — điều kiện chạy

1. `$AUTO_FIX` phải là `true`. Nếu không, skip toàn bộ workflow này, không in gì thêm.
2. `$IS_GIT_REPO` phải là `true`. Nếu không có git, KHÔNG có cách revert an toàn khi build fail → in `{msg_autofix_needs_git}` rồi skip (không chạm file nào).
3. Nếu repo có uncommitted changes NGOÀI scope đang scan, in cảnh báo `{msg_autofix_dirty_tree}` một lần (khuyến nghị commit/backup trước) nhưng vẫn tiếp tục — user đã explicit opt-in qua flag.
4. Nếu `$SCAN_ROOT` khác `.`: file đang quét là snapshot tạm của commit/PR (bị `rm -rf` sau khi render report), không phải working tree — sửa ở đó không có tác dụng. In `{msg_autofix_snapshot_scope}` một lần, rồi với mọi finding CRITICAL/HIGH: harvest context (Bước 1, đọc tại `$SCAN_ROOT/<file>`) → generate diff (Bước 2) → đi thẳng Bước 4 (ghi patch, `patch_status: "suggested_only"`). KHÔNG `git apply`, KHÔNG build verify. Chỉ khi `$SCAN_ROOT` là `.` mới chạy Bước 3.

## Scope — finding nào được auto-fix

- Chỉ xử lý finding có `severity` là `CRITICAL` hoặc `HIGH`. MEDIUM/LOW → `patch_status: "skipped_low_severity"`, không đụng vào (rủi ro/nhiễu không tương xứng với lợi ích tự động sửa).
- Xử lý **tuần tự**, nhóm theo `file`: nếu 1 file có nhiều finding, xử lý hết các finding của file đó trước khi qua file khác (tránh 2 patch chồng lấn cùng lúc trên 1 file).
- Bỏ qua finding đã có `patch_status` (vd re-run sau khi 1 phần đã fix) trừ khi user chỉ định re-run.

## Bước 1 — Context Harvesting (per finding)

1. **Read** toàn bộ file chứa finding (tại `$SCAN_ROOT/<file>`) (nếu file lớn, Read đoạn `[max(1, line-25), line+25]`).
2. Lấy thêm **import/using/require block** ở đầu file (thường 1-30 dòng đầu, dừng khi gặp dòng code thực sự đầu tiên) — cần để agent biết namespace/dependency đã có sẵn, tránh sinh patch dùng thư viện chưa import.
3. Với finding có `rule_id` trỏ tới rule có override ở `rules/languages/<lang>/`, đọc phần **`## Fix recommendation`** của rule đó (generic hoặc overlay) — đây là pattern fix chuẩn, dùng làm khung cho patch.

## Bước 2 — Patch Generation

Yêu cầu nghiêm ngặt: agent CHỈ xuất ra **unified diff**, không kèm giải thích, không code block khác xen vào giữa.

Format bắt buộc:

````
```diff
--- a/path/to/file.ext
+++ b/path/to/file.ext
@@ -<start>,<count> +<start>,<count> @@
 context line
-old line
+new line
 context line
```
````

Quy tắc:
- Path trong `--- a/` và `+++ b/` PHẢI tương đối với git root, giống format `git diff` thật.
- Hunk phải chứa ít nhất 3 dòng context mỗi phía (trừ khi finding ở đầu/cuối file) để `git apply` match được vị trí chính xác.
- Patch chỉ sửa đúng phần liên quan đến finding — không "nhân tiện" refactor code xung quanh.
- Nếu finding là `rule_id: VULNERABLE-DEPENDENCY` (từ `--sca`, xem [`../references/dependency-scan.md`](../references/dependency-scan.md)), patch sửa **version trong manifest** (`.csproj`, `package.json`, `go.mod`, `composer.json`, `requirements.txt`...), không sửa code logic.

## Bước 3 — Verification Loop

Đây là bước bắt buộc trước khi patch được giữ lại vĩnh viễn. Thực hiện qua Bash tool:

1. **Trial apply (tương đương "áp dụng vào file tạm"):**
   ```bash
   git apply --check <patch-file>
   ```
   Đây là dry-run — không chạm file thật. Nếu fail (patch không match context, path sai...) → **không tính vào retry budget** (đây là lỗi format, không phải lỗi build) → agent regenerate diff với context chính xác hơn, thử lại `--check`.

2. **Apply thật** (file đã git-tracked → luôn revert được):
   ```bash
   git apply <patch-file>
   ```

3. **Verify build** — chọn command theo `$PRIMARY_LANG` (khớp `references/language-detection.md`):

   | Lang | Verify command | Ghi chú |
   |---|---|---|
   | `dotnet` | `dotnet build` | Chạy ở thư mục chứa `.sln` gần nhất, hoặc `.csproj` nếu không có `.sln` |
   | `go` | `go build ./...` | Chạy ở module root (`go.mod`) |
   | `typescript` | `npx tsc --noEmit` nếu có `tsconfig.json`, else `node --check <file>` | JS thuần không có type-check dự án, chỉ check syntax file |
   | `php` | `php -l <file>` | Chỉ syntax check, không semantic toàn dự án |
   | `python` | `python -m py_compile <file>` | Chỉ syntax check, không semantic toàn dự án |
   | *(không có overlay / lang khác)* | — | Không có verify command tin cậy → KHÔNG auto-apply, xem "Không verify được" dưới đây |

   **Đặc biệt — patch sửa dependency (`VULNERABLE-DEPENDENCY`):** phải chạy lại package manager trước khi build, vì version mới cần resolve lockfile:
   - dotnet: `dotnet restore && dotnet build`
   - npm: `npm install && (npx tsc --noEmit || npm run build)`
   - Go: `go get <package>@<fixed_version> && go build ./...`
   - Composer: `composer update <package> && composer validate`
   - pip: `pip install -U <package>==<fixed_version> && python -m py_compile <file>`

4. **Build thành công** → giữ patch, `patch_status: "applied"`, qua finding tiếp theo.

5. **Build fail** →
   ```bash
   git checkout -- <file>
   ```
   Revert về bản gốc. Lấy ~50 dòng cuối của stderr/stdout, đưa vào prompt sinh patch lần sau (kèm nguyên context ở Bước 1). Quay lại Bước 2.

6. **Retry budget: tối đa 2 lần thử lại** (tổng 3 lần generate: 1 lần đầu + 2 retry). Hết budget mà vẫn fail → `patch_status: "failed_verification"`, giữ nguyên file gốc, để user tự sửa tay.

## Bước 4 — Không verify được (lang không có build command)

Khi `$PRIMARY_LANG` không có verify command tin cậy (rule "không có overlay" ở bảng trên, hoặc ngôn ngữ chưa được vbsec chuyên sâu hoá):

1. KHÔNG áp dụng patch vào file thật.
2. Ghi diff ra file: `vbsec-reports/patches/<file-slug>-<line>.patch` (dùng Write tool).
3. `patch_status: "suggested_only"`.

## Bước 5 — Reporting

Mỗi finding đã qua auto-fix cần thêm field `patch_status` vào entry tương ứng trong `findings[]` của JSON canonical (giá trị: `applied` | `failed_verification` | `suggested_only` | `skipped_low_severity`).

Thêm 1 section Markdown mới vào report, đặt **trước** JSON summary (sau section "Next steps", trước Footer) — dùng key i18n:

```markdown
## {header_autofix_title}

| # | {col_file_line} | {col_rule} | Trạng thái |
|---|---|---|---|
| 1 | `api/users.ts:42` | SQL-INJECTION | ✅ {autofix_status_applied} |
| 2 | `auth.ts:18` | WEAK-PASSWORD-HASHING | ❌ {autofix_status_failed} |
| 3 | `pkg.csproj:5` | VULNERABLE-DEPENDENCY | 📄 {autofix_status_suggested} |
```

`{msg_autofix_summary}`: 1 dòng tổng kết (vd "Đã tự sửa 5/8 lỗi CRITICAL+HIGH. 2 lỗi cần sửa tay, 1 lỗi chỉ có gợi ý patch (xem vbsec-reports/patches/).").

Các i18n key mới (đã có template ở `references/i18n/{vi,en}.md`):
`header_autofix_title`, `msg_autofix_needs_git`, `msg_autofix_dirty_tree`, `autofix_status_applied`, `autofix_status_failed`, `autofix_status_suggested`, `autofix_status_skipped`, `msg_autofix_summary`, `msg_autofix_snapshot_scope`.

## Edge cases

| Scenario | Xử lý |
|---|---|
| Nhiều finding cùng file, patch sau đè context của patch trước | Xử lý tuần tự, re-Read file sau mỗi lần apply thành công để lấy context mới nhất trước khi patch finding tiếp theo cùng file |
| `git apply --check` fail liên tục do context lệch | Không tính vào retry budget, nhưng nếu fail quá 3 lần dry-run (bug logic, không phải build fail) → bỏ qua finding, `patch_status: "failed_verification"`, note lý do "patch không apply được" |
| Build command không tồn tại trên máy (vd không có `dotnet` CLI) | Coi như build fail ngay từ lần đầu (không có cách verify) → xử lý như Bước 4 (suggested_only) thay vì retry vô ích |
| Finding trong file đã bị xoá/rename từ lúc scan tới lúc auto-fix | Skip, note "file không còn tồn tại" |
| User không có `.gitignore` cho `vbsec-reports/patches/` | Dùng lại `$GITIGNORE_WARNING` đã có từ SKILL.md Step 0, không cần check riêng |
