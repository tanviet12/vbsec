# Auto-fix Workflow (`--auto-fix`)

Vòng lặp patch tự động cho findings CRITICAL/HIGH, chạy sau khi đã có `findings[]` và trước khi render report.

> Được gọi từ [`../SKILL.md`](../SKILL.md) **Step 4c** khi user truyền flag `--auto-fix` — chạy TRƯỚC khi report cuối được render ở Step 5, để `patch_status` có mặt trong JSON summary và section Auto-fix. File này giả định `findings[]` (canonical, xem [`../references/output-format.md`](../references/output-format.md)) đã có từ Step 4 (+ Step 4b nếu có `--sca`).

**Đây là bước duy nhất trong skill được phép ghi đè file nguồn của user.** Mọi bước khác của vbsec chỉ đọc + báo cáo.

**Nguyên tắc an toàn:** mọi file có thể bị ghi (file bị patch, manifest, lockfile) phải được snapshot TRƯỚC khi ghi, và khi fail thì khôi phục từ snapshot. KHÔNG dùng `git checkout -- <file>` / `git restore` để revert: các lệnh đó đưa file về bản trong index/HEAD và xoá mất thay đổi chưa commit của user (đúng trường hợp `uncommitted --auto-fix`), còn với file untracked thì không revert được.

## Inputs

- `$AUTO_FIX` — `true` nếu flag được truyền (xem SKILL.md Step 0)
- `$IS_GIT_REPO` — bắt buộc `true` để chạy bước này
- `$SCAN_ROOT` — `.` khi quét working tree; thư mục snapshot tạm khi scope là `commit id` / `pr id` (xem SKILL.md Step 0)
- `$PRIMARY_LANG` — dùng để chọn verify command
- `findings[]` từ Step 4 (+ Step 4b) (mỗi finding có `file`, `line`, `rule_id`, `severity`)

## Gate — điều kiện chạy

1. `$AUTO_FIX` phải là `true`. Nếu không, skip toàn bộ workflow này, không in gì thêm.
2. `$IS_GIT_REPO` phải là `true` (patch được kiểm tra và apply bằng `git apply`). Nếu không → in `{msg_autofix_needs_git}` rồi skip (không chạm file nào).
3. Nếu repo có uncommitted changes NGOÀI scope đang scan, in cảnh báo `{msg_autofix_dirty_tree}` một lần (khuyến nghị commit/backup trước) nhưng vẫn tiếp tục — user đã explicit opt-in qua flag.
4. Nếu `$SCAN_ROOT` khác `.`: file đang quét là snapshot tạm của commit/PR (bị `rm -rf` sau khi render report), không phải working tree — sửa ở đó không có tác dụng. In `{msg_autofix_snapshot_scope}` một lần, rồi với mọi finding CRITICAL/HIGH: harvest context (Bước 1, đọc tại `$SCAN_ROOT/<file>`) → generate diff (Bước 2) → đi thẳng Bước 4 (ghi patch, `patch_status: "suggested_only"`). KHÔNG `git apply`, KHÔNG build verify. Chỉ khi `$SCAN_ROOT` là `.` mới chạy Bước 0 và Bước 3.

## Scope — finding nào được auto-fix

- Chỉ xử lý finding có `severity` là `CRITICAL` hoặc `HIGH`. MEDIUM/LOW → `patch_status: "skipped_low_severity"`, không đụng vào (rủi ro/nhiễu không tương xứng với lợi ích tự động sửa).
- Xử lý **tuần tự**, nhóm theo `file`: nếu 1 file có nhiều finding, xử lý hết các finding của file đó trước khi qua file khác (tránh 2 patch chồng lấn cùng lúc trên 1 file).
- Bỏ qua finding đã có `patch_status` (vd re-run sau khi 1 phần đã fix) trừ khi user chỉ định re-run.

## Bước 0 — Preflight (1 lần, TRƯỚC khi apply bất kỳ patch nào)

Chạy khi `$SCAN_ROOT` là `.`. Mục đích: biết chắc có verify được hay không trước khi chạm vào file thật.

1. **Build tool có tồn tại không.** Kiểm tra bằng `command -v` cho tool của `$PRIMARY_LANG` (bảng ở Bước 3):
   ```bash
   command -v dotnet >/dev/null 2>&1 || VERIFY_TOOL_MISSING=true   # vd lang = dotnet
   ```
   Thiếu tool → KHÔNG apply patch nào: mọi finding CRITICAL/HIGH đi thẳng Bước 4 (`suggested_only`), note "thiếu `<tool>`, không verify được".
   Với finding `VULNERABLE-DEPENDENCY` của Go/dotnet, kiểm tra thêm `go` / `dotnet` (bảng "Patch sửa dependency" ở Bước 3). Thiếu → finding đó đi thẳng Bước 4, không apply vào manifest.
2. **Build baseline.** Chạy verify command 1 lần trên code hiện tại, chưa patch gì:
   - Verify cấp dự án (`dotnet build`, `go build ./...`, `npx tsc --noEmit`): chạy 1 lần. Fail → project vốn đã build lỗi, không phân biệt được lỗi do patch hay lỗi có sẵn → mọi finding đi thẳng Bước 4 (`suggested_only`), note "build baseline fail".
   - Verify cấp file (`php -l`, `python -m py_compile`, `node --check`): chạy trên từng file trước khi patch file đó. Fail → finding của file đó đi thẳng Bước 4.
3. Tạo thư mục snapshot dùng chung cho cả lượt auto-fix (nằm ngoài repo):
   ```bash
   AF_BAK=$(mktemp -d "${TMPDIR:-/tmp}/vbsec-autofix.XXXXXX")
   ```
   Xoá `$AF_BAK` sau khi xong toàn bộ workflow (kể cả khi dừng giữa chừng).

## Bước 1 — Context Harvesting (per finding)

1. **Read** toàn bộ file chứa finding tại `$SCAN_ROOT/<file>` (nếu file lớn, Read đoạn `[max(1, line-25), line+25]`).
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
- Nếu finding là `rule_id: VULNERABLE-DEPENDENCY` (từ `--sca`, xem [`../references/dependency-scan.md`](../references/dependency-scan.md)), patch sửa **version trong manifest** (`.csproj`, `package.json`, `go.mod`, `composer.json`...), không sửa code logic. Xem mục "Patch sửa dependency" ở Bước 3 để biết ecosystem nào được apply.

## Bước 3 — Verification Loop

Đây là bước bắt buộc trước khi patch được giữ lại vĩnh viễn. Chỉ chạy khi `$SCAN_ROOT` là `.` và Bước 0 không chuyển finding sang Bước 4. Thực hiện qua Bash tool:

1. **Dry-run:**
   ```bash
   git apply --check <patch-file>
   ```
   Không chạm file thật. Nếu fail (patch không match context, path sai...) → **không tính vào retry budget** (đây là lỗi format, không phải lỗi build) → agent regenerate diff với context chính xác hơn, thử lại `--check`.

2. **Snapshot** mọi file sắp bị ghi: file trong patch, cộng manifest + lockfile nếu là patch dependency (danh sách ở mục dưới). File chưa tồn tại thì ghi tên vào danh sách `absent` để khi khôi phục thì xoá đi:
   ```bash
   for f in $SNAPSHOT_FILES; do
     if [ -e "$f" ]; then
       mkdir -p "$AF_BAK/$(dirname "$f")" && cp -p "$f" "$AF_BAK/$f"
     else
       echo "$f" >> "$AF_BAK/.absent"
     fi
   done
   ```
   Snapshot lưu nội dung ĐÚNG lúc trước khi patch, gồm cả thay đổi chưa commit và file untracked.

3. **Apply thật:**
   ```bash
   git apply <patch-file>
   ```

4. **Verify build** — chọn command theo `$PRIMARY_LANG` (khớp `references/language-detection.md`):

   | Lang | Tool kiểm tra ở Bước 0 | Verify command | Ghi chú |
   |---|---|---|---|
   | `dotnet` | `dotnet` | `dotnet build` | Chạy ở thư mục chứa `.sln` gần nhất, hoặc `.csproj` nếu không có `.sln` |
   | `go` | `go` | `go build ./...` | Chạy ở module root (`go.mod`) |
   | `typescript` | `npx` + `tsc` trong project (có `tsconfig.json`), else `node` | `npx --no-install tsc --noEmit` nếu có `tsconfig.json`, else `node --check <file>` | `--no-install` để không tự tải `tsc` từ registry. JS thuần chỉ check syntax file |
   | `php` | `php` | `php -l <file>` | Chỉ syntax check, không semantic toàn dự án |
   | `python` | `python3` hoặc `python` | `python3 -m py_compile <file>` | Chỉ syntax check, không semantic toàn dự án |
   | *(không có overlay / lang khác)* | — | — | Không có verify command tin cậy → KHÔNG auto-apply, xem Bước 4 |

   **Patch sửa dependency (`VULNERABLE-DEPENDENCY`)** — build pass chỉ chứng minh version mới resolve được, không chứng minh code vẫn tương thích (bản mới có thể có breaking change). Vì vậy bump dependency chỉ được `applied` khi **test của project** chạy pass trên version mới. Ecosystem nào không build + test được với version mới thì chỉ gợi ý patch:

   | Ecosystem | Snapshot thêm | Resolve + build + test | Kết quả |
   |---|---|---|---|
   | Go | `go.mod`, `go.sum` | `go get <module>@<fixed_version> && go build ./... && go test ./...` | Test pass → `applied` |
   | dotnet | `.csproj` bị patch, `packages.lock.json`, `Directory.Packages.props` (nếu có) | `dotnet restore && dotnet build && dotnet test` | Test pass → `applied`. Khi khôi phục, chạy lại `dotnet restore` để `obj/` khớp manifest cũ |
   | npm, Composer | — | Không chạy. Muốn test phải cài package vào `node_modules/`/`vendor/` và chạy install script — không hoàn tác gọn được | Luôn Bước 4 (`suggested_only`) |
   | PyPI | — | KHÔNG chạy `pip install` (sửa thẳng môi trường Python của user, không hoàn tác được) | Luôn Bước 4 (`suggested_only`) |

   Quy tắc cho Go/dotnet:
   - **Project không có test** (Go: không có file `*_test.go`; dotnet: không có project nào tham chiếu `Microsoft.NET.Test.Sdk`) → không verify được tương thích → Bước 4 (`suggested_only`), không apply.
   - **Test baseline:** chạy `go test ./...` / `dotnet test` 1 lần TRƯỚC khi bump (cùng lúc với build baseline ở Bước 0, chỉ khi có finding dependency của Go/dotnet). Baseline fail → không phân biệt được lỗi do bump hay lỗi có sẵn → Bước 4.
   - **Resolve, build hoặc test fail sau khi bump** → khôi phục từ snapshot (mục 6 bên dưới), `patch_status: "failed_verification"`, note "bản <fixed_version> không tương thích, cần sửa code khi nâng cấp". KHÔNG retry: patch bump version là cố định, sinh lại cũng ra đúng patch đó.

5. **Build thành công** → giữ patch, `patch_status: "applied"`, xoá snapshot của các file này khỏi `$AF_BAK`, qua finding tiếp theo.

6. **Build fail** → khôi phục từ snapshot (KHÔNG dùng `git checkout`):
   ```bash
   for f in $SNAPSHOT_FILES; do
     if grep -qxF "$f" "$AF_BAK/.absent" 2>/dev/null; then
       rm -f "$f"
     else
       cp -p "$AF_BAK/$f" "$f"
     fi
   done
   ```
   Sau khi khôi phục, so sánh lại (`cmp`) từng file với snapshot để chắc chắn đã về đúng bản trước patch. Lấy ~50 dòng cuối của stderr/stdout, đưa vào prompt sinh patch lần sau (kèm nguyên context ở Bước 1). Quay lại Bước 2.

7. **Retry budget: tối đa 2 lần thử lại** (tổng 3 lần generate: 1 lần đầu + 2 retry). Hết budget mà vẫn fail → `patch_status: "failed_verification"`, file giữ nguyên như trước khi auto-fix chạy, để user tự sửa tay.

## Bước 4 — Không apply (chỉ gợi ý patch)

Các trường hợp: `$SCAN_ROOT` khác `.` (Gate 4); `$PRIMARY_LANG` không có verify command tin cậy; thiếu build tool hoặc build baseline fail (Bước 0); dependency npm/Composer/PyPI, hoặc dependency Go/dotnet khi project không có test hoặc test baseline fail (Bước 3).

1. KHÔNG áp dụng patch vào file thật.
2. Ghi diff ra file: `vbsec-reports/patches/<file-slug>-<line>.patch` (dùng Write tool).
3. `patch_status: "suggested_only"`, kèm lý do ngắn trong report (vd "thiếu `dotnet`", "build baseline fail", "dependency npm: không verify được tương thích", "không có test").

## Bước 5 — Reporting

Mỗi finding đã qua auto-fix cần thêm field `patch_status` vào entry tương ứng trong `findings[]` của JSON canonical (giá trị: `applied` | `failed_verification` | `suggested_only` | `skipped_low_severity`).

Thêm 1 section Markdown mới vào report, đặt **trước** JSON summary (sau section "Next steps", trước Footer) — dùng key i18n:

```markdown
## {header_autofix_title}

| # | {col_file_line} | {col_rule} | Trạng thái |
|---|---|---|---|
| 1 | `api/users.ts:42` | SQL-INJECTION | ✅ {autofix_status_applied} |
| 2 | `auth.ts:18` | WEAK-PASSWORD-HASHING | ❌ {autofix_status_failed} |
| 3 | `package.json:12` | VULNERABLE-DEPENDENCY | 📄 {autofix_status_suggested} |
```

`{msg_autofix_summary}`: 1 dòng tổng kết (vd "Đã tự sửa 5/8 lỗi CRITICAL+HIGH. 2 lỗi cần sửa tay, 1 lỗi chỉ có gợi ý patch (xem vbsec-reports/patches/).").

Các i18n key mới (đã có template ở `references/i18n/{vi,en}.md`):
`header_autofix_title`, `msg_autofix_needs_git`, `msg_autofix_dirty_tree`, `autofix_status_applied`, `autofix_status_failed`, `autofix_status_suggested`, `autofix_status_skipped`, `msg_autofix_summary`, `msg_autofix_snapshot_scope`.

## Edge cases

| Scenario | Xử lý |
|---|---|
| Nhiều finding cùng file, patch sau đè context của patch trước | Xử lý tuần tự, re-Read file sau mỗi lần apply thành công để lấy context mới nhất trước khi patch finding tiếp theo cùng file. Snapshot chụp lại trước mỗi patch, nên fail ở patch sau chỉ gỡ patch sau, patch trước đã `applied` vẫn giữ |
| `git apply --check` fail liên tục do context lệch | Không tính vào retry budget, nhưng nếu fail quá 3 lần dry-run (bug logic, không phải build fail) → bỏ qua finding, `patch_status: "failed_verification"`, note lý do "patch không apply được" |
| Build command không tồn tại trên máy (vd không có `dotnet` CLI) | Phát hiện ở Bước 0 bằng `command -v`, TRƯỚC khi apply → Bước 4 (`suggested_only`), file không bị sửa |
| Project vốn đã build lỗi | Phát hiện ở Bước 0 (build baseline) → Bước 4 (`suggested_only`) thay vì apply/revert 3 vòng vô ích |
| Finding trong file đã bị xoá/rename từ lúc scan tới lúc auto-fix | Skip, note "file không còn tồn tại" |
| Workflow bị dừng giữa chừng (user huỷ, lỗi tool) | Trước khi thoát, khôi phục mọi file còn nằm trong `$AF_BAK` (patch chưa verify xong), rồi xoá `$AF_BAK` |
| User không có `.gitignore` cho `vbsec-reports/patches/` | Dùng lại `$GITIGNORE_WARNING` đã có từ SKILL.md Step 0, không cần check riêng |
