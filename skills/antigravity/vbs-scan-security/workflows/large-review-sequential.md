# LARGE Review Workflow — Sequential (Codex/Antigravity)

Sequential chunking workflow cho repo lớn (>20 main-lang files HOẶC >30 total HOẶC >14 ngày). Khác với Claude Code variant (spawn parallel sub-agents), variant này xử lý **từng chunk tuần tự** trong cùng main agent context.

> Được gọi từ [`../SKILL.md`](../SKILL.md) Step 3 khi routing quyết định LARGE mode.

## Tại sao sequential?

Codex CLI và Antigravity (tại thời điểm này) chưa có cơ chế cho skill spawn parallel sub-agents. Để giữ output IDENTICAL với Claude Code variant, ta chia file thành chunks (giống Claude) nhưng xử lý lần lượt. Trade-off: chậm hơn ~3× nhưng kết quả y nguyên.

## Inputs (đã có sẵn từ SKILL.md context)

- `$SCOPE`, `$LANG`, `$FILES`, `$SCAN_ROOT`, `$PRIMARY_LANG`, `$OVERLAY_AVAILABLE`, i18n strings — same as SMALL mode

## Steps

### Step L1 — Load rule files (1 lần duy nhất)

Nạp phần phát hiện của bộ rule vào context **1 lần ở đầu workflow** để tránh re-read mỗi chunk:

```bash
bash <skill-dir>/references/load-rules.sh --part 1 <primary_lang> [<lang2> ...]
```

(`<skill-dir>` = thư mục chứa `SKILL.md`. Lang không có overlay → bỏ tham số lang.) Dòng cuối output cho biết tổng số phần (`PART 1/4`...) → chạy tiếp `--part 2`, `--part 3`... (chạy song song được) tới khi thấy "đã in đủ". Output chia phần vì Bash tool cắt output dài; **KHÔNG pipe qua `head`/`tail`**. Gộp lại, script in đủ 21 rule, mỗi rule chỉ gồm phần phát hiện: Intent, điều kiện severity, cách reasoning, search patterns. Rule có overlay cho lang đã detect → overlay đã thay generic (repo đa ngôn ngữ: in cả generic lẫn overlay khi chưa phải lang nào cũng có overlay). **KHÔNG Read lại từng file rule ở bước này.**

Phần chi tiết (Examples, Fix recommendation) chỉ Read khi rule có finding CRITICAL/HIGH, lúc render report.

### Step L2 — Setup workspace

```bash
mkdir -p .vbsec-tmp
```

Đảm bảo `.vbsec-tmp/` trong `.gitignore` (warn user nếu không, nhưng vẫn proceed).

### Step L3 — Chunk files

Đọc [`../references/chunking-strategy.md`](../references/chunking-strategy.md) và apply algorithm.

Output: list `chunks` với format:
```
chunks = [
  {"name": "api/handlers", "slug": "api-handlers", "files": [...], "count": 12},
  {"name": "frontend/src/components", "slug": "frontend-components", "files": [...], "count": 25},
  ...
]
```

`slug` = `name` với `/` thay bằng `-`.

### Step L4 — Process chunks SEQUENTIALLY

For mỗi `chunk` trong `chunks` (theo thứ tự, không parallel):

1. **Resume check:** nếu `.vbsec-tmp/findings-<slug>.md` đã tồn tại và non-empty → skip chunk này (đã scan ở session trước). Đọc lại file vào memory.

2. **Print progress** ra stdout: `[chunk N/total] Scanning <chunk.name> (<count> files)...`

3. **Apply rules cho files trong chunk:**
   - Cho mỗi file trong `chunk.files`:
     - Skip nếu binary/generated/>5000 dòng (xem [`small-review.md`](small-review.md#step-s2--apply-rules-per-file) Step S2.1)
     - Cho mỗi rule trong (generic + overlay):
       - Dùng grep tool để tìm patterns
       - Dùng read tool để xem full function/context
       - Apply L1-L4 data flow analysis (xem [`../references/data-flow-classification.md`](../references/data-flow-classification.md))
       - Quyết định: vulnerability thật hay false positive?
   - Collect findings: `(file, line, rule_id, severity, issue, fix, context)`

4. **Rule ID discipline (BẮT BUỘC):**
   - **Chỉ dùng 21 canonical rule IDs**. KHÔNG tự bịa rule mới.
   - 1 dòng code dính 2 rule → tạo **2 finding riêng biệt**, mỗi cái 1 `rule_id`.

5. **Write chunk findings** vào `.vbsec-tmp/findings-<slug>.md`:
   - Format markdown (cùng schema với sub-agent output của Claude variant)
   - Sections: `## FINDINGS`, `## PASSED`, `## NOT_MAPPED` (nếu có), `## HARDENING_NOTES` (nếu có — gợi ý phòng thủ, không phải finding)
   - File path tuyệt đối từ repo root

6. **Print confirmation:** `[chunk N/total] ✓ <count_findings> findings`

7. **Tiếp tục chunk tiếp theo** (không spawn, không await — vì đã sequential).

### Step L5 — Aggregate findings

Đọc tất cả `.vbsec-tmp/findings-*.md`:

1. **Parse** mỗi file thành list findings (file/line/rule_id/severity/issue/fix/context)
2. **Validate rule_ids**: mọi finding phải có `rule_id` trong 21 canonical IDs.
3. **Dedup**: key = `(file, line, rule_id)`. Giữ entry có severity cao nhất. Nếu tie, giữ entry có `context` dài hơn.
   - **Lưu ý:** dedup key có `rule_id` → 1 vị trí (file:line) dính 2 rule khác nhau (vd IDOR + RACE) sẽ là 2 entry riêng, KHÔNG dedup.
4. **Collect HARDENING_NOTES**: gộp vào `hardening_notes[]` + section `{header_hardening_title}` (dedup, tối đa 5 dòng). KHÔNG đưa vào `findings[]`.
5. **Collect NOT_MAPPED**: nếu có, note ở cuối main report (giúp roadmap future rules).
6. **Collect PASSED**: union các rule_id xuất hiện trong `## PASSED` section của tất cả chunks. Một rule chỉ vào PASSED list nếu **không** xuất hiện trong findings của bất kỳ chunk nào.
7. **Cross-chunk rules**:
   - **SLOPSQUATTING**: collect tất cả import statement từ chunks, dedup, kiểm tra package có hợp lệ.
   - **OUTDATED-DEPENDENCY**: đọc file dependency lock (`package-lock.json`, `go.sum`, `composer.lock`) ở root.
   - **CSRF middleware global**: nếu phát hiện middleware global ở 1 chunk, downgrade các CSRF finding ở chunk khác.

8. **Counts sanity check** (BẮT BUỘC trước khi render):
   ```
   total = len(findings)
   assert total == count_by_severity('CRITICAL') + count_by_severity('HIGH') + count_by_severity('MEDIUM') + count_by_severity('LOW')
   ```

### Step L6 — Translate (if lang=vi)

Nếu `$LANG = "vi"`:

Với mỗi finding:
- `issue` (EN) → translate sang vi, giữ technical terms tiếng Anh (function name, library, code snippet)
- `fix` (EN) → ưu tiên dùng phrase template từ `i18n/vi.md`

Section headers, verdict labels — lấy từ i18n key đã load.

### Step L7 — Render report

Theo template trong [`../references/output-format.md`](../references/output-format.md).

**Verbose level theo severity:**
- CRITICAL → overview table + full verbose block (Mô tả ngắn + Tại sao nguy hiểm + Attack scenario + Code before/after + Đọc thêm)
- HIGH → overview table + medium block (Mô tả + Tác động + Code fix + Đọc thêm)
- MEDIUM → compact table only
- LOW → compact table only

**Generate verbose content (non-tech friendly):**

Findings từ chunks là compact (file, line, rule_id, severity, issue, fix, context). Khi render, paraphrase từ rule file content (đã Read ở Step L1):
- `why_dangerous` → từ section "Intent" + "Examples CRITICAL" của rule
- `attack_scenario` → kịch bản thực tế dựa trên rule's pattern + chunk's `context`
- `code_before` → từ chunk's `context` (đã có snippet thực)
- `code_after` → từ section "Fix recommendation" của rule, adapt cho code thực tế

Khi translate sang `$LANG=vi`: dịch text, giữ code English. Khi `lang=en`: dùng EN canonical.

### Step L8 — Save report to file

1. Render full report (theo Step L7 logic)
2. **Save** dùng write/create-file tool của Antigravity:
   - Path: `vbsec-reports/scan-<TIMESTAMP>.md` (đã prepare ở SKILL.md Step 0)
   - Nội dung IDENTICAL với stdout
3. **Print stdout** sau report:
   ```
   📄 {msg_report_saved}: vbsec-reports/scan-2026-05-13-143022.md
   ```
4. **If gitignore warning needed:**
   ```
   ⚠️ {msg_gitignore_warning_title}: {msg_gitignore_warning_text}
   ```

### Step L9 — Determine verdict

| Findings | Verdict |
|---|---|
| ≥1 CRITICAL | FAIL |
| 0 CRITICAL, ≥1 HIGH | WARN |
| 0 CRITICAL, 0 HIGH | PASS |

### Step L10 — Cleanup

```bash
rm -rf .vbsec-tmp    # cleanup temp files (luôn xóa)
# KHÔNG xóa vbsec-reports/ — đó là persisted output cho user
```

## Resume protocol

Nếu user re-run skill khi `.vbsec-tmp/` còn từ session trước:

1. Đọc `.vbsec-tmp/` — chunks nào đã có `findings-*.md` non-empty thì coi như đã scan
2. Chỉ process chunks chưa có findings file (Step L4 đã có resume check ở mục 1)
3. Aggregate như bình thường (Step L5)

Nếu user muốn re-scan từ đầu: dùng arg `--fresh` → xóa `.vbsec-tmp/` trước khi bắt đầu.

## Performance target

| Chunks | Sequential time |
|---|---|
| 5 | ~5-8 min |
| 10 | ~10-15 min |
| 15 | ~15-25 min |

So sánh: Claude variant parallel ~2-10 min cho cùng workload. Sequential variant chậm hơn ~3× nhưng output identical.

Main agent context: ~50-100K tokens (đủ cho repo trung bình; nếu repo cực lớn → có thể cần chia thành 2 lần invoke).

## Edge cases

| Scenario | Handling |
|---|---|
| Chunk có 1 file rất lớn (>5000 dòng) | Đọc bằng grep trước → read targeted sections |
| 2 chunks cùng tìm thấy lỗi trong file giống nhau (file ở biên) | Dedup ở Step L5 |
| Generated code chiếm cả 1 chunk | Flag tự "this chunk is mostly generated, low priority". Giảm severity các finding trong chunk này 1 cấp. |
| Repo 500+ file → 15 chunk, mỗi chunk 30+ file | OK nhưng chậm (~20-30 min). Cân nhắc gợi ý user dùng Claude Code variant cho repo cỡ này. |
| User Ctrl+C giữa chừng | `.vbsec-tmp/` giữ lại. Re-run → resume từ chunk dở dang. |
| Không có git (repo chưa init) | Lỗi sớm ở SKILL.md Step 0 — không vào đây. |
| Context của agent gần đầy giữa chừng | Save partial findings vào `.vbsec-tmp/` rồi báo user re-invoke skill để continue. |
