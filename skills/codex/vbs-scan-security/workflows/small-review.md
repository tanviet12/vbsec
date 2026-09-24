# SMALL Review Workflow

Inline scan workflow cho repo nhỏ-vừa (≤20 main-lang files VÀ ≤30 total VÀ ≤14 ngày). Main agent tự chạy hết, không spawn sub-agent.

> Được gọi từ [`../SKILL.md`](../SKILL.md) Step 3 khi routing quyết định SMALL mode. Trước khi đọc file này, SKILL.md đã làm: parse args, gather files, detect lang, load i18n.

## Inputs (đã có sẵn từ SKILL.md context)

- `$SCOPE` — scope label
- `$LANG` — output lang (`vi` hoặc `en`)
- `$FILES` — list file đã filter (loại vendored)
- `$SCAN_ROOT` — thư mục chứa code cần đọc (`.` hoặc snapshot tạm cho scope `commit id`/`pr id`). Đọc file tại `$SCAN_ROOT/<path>`, report ghi `<path>`
- `$PRIMARY_LANG` — kết quả detect language
- `$OVERLAY_AVAILABLE` — có rule chuyên sâu không
- i18n strings đã load (từ `references/i18n/<lang>.md`)

## Steps

### Step S1 — Load applicable rules

1. **Nạp phần phát hiện của bộ rule hiệu lực qua script (vài lệnh Bash, mỗi lệnh 1 phần):**

   ```bash
   bash <skill-dir>/references/load-rules.sh --part 1 <primary_lang> [<lang2> ...]
   ```

   (`<skill-dir>` = thư mục chứa `SKILL.md`. Lang không có overlay → bỏ tham số lang.) Dòng cuối output cho biết tổng số phần (`PART 1/4`...) → chạy tiếp `--part 2`, `--part 3`... (chạy song song được) tới khi thấy "đã in đủ". Output chia phần vì Bash tool cắt output dài; **KHÔNG pipe qua `head`/`tail`**. Gộp lại, script in đủ 21 rule, mỗi rule chỉ gồm phần phát hiện: Intent, điều kiện severity, cách reasoning, search patterns. Rule có overlay cho lang đã detect → overlay đã thay generic (repo đa ngôn ngữ: in cả generic lẫn overlay khi chưa phải lang nào cũng có overlay). **KHÔNG Read lại từng file rule ở bước này.**

2. **Phần chi tiết (Examples, Fix recommendation, Cross-references) chỉ đọc khi cần:**
   - Rule có finding **CRITICAL/HIGH** → Read file theo path ở header `=== RULE <id> (source: ...) ===` để lấy code fix và ví dụ cho verbose block trong report.
   - Phân vân một match là lỗi thật hay false positive → được Read phần Examples của rule đó để đối chiếu ví dụ an toàn / không an toàn.
   - Finding MEDIUM/LOW chỉ cần bảng compact → không cần đọc phần chi tiết.

### Step S2 — Apply rules per file

Cho mỗi file trong `$FILES`:

1. **Skip nếu không applicable**:
   - File binary (image, font, archive) → skip
   - File generated (vd: `*.pb.go`, `*_pb2.py`, `dist/*`) → skip với note
   - File >5000 dòng → đọc nhưng có thể dùng search-then-read pattern (Grep tìm pattern, Read context xung quanh)

2. **Với mỗi rule applicable**:
   - Đọc `Search patterns` của rule (chỉ là gợi ý, không chạy literal)
   - Dùng **Grep tool** để tìm patterns trong file
   - Với mỗi match, **Read tool** để xem full function/context
   - Áp dụng **L1-L4 data flow analysis** (xem [`../references/data-flow-classification.md`](../references/data-flow-classification.md))
   - Quyết định: vulnerability thật hay false positive?

3. **Ghi nhận finding** vào memory với fields:
   - `file`, `line`, `rule_id`, `severity` (≤ `severity_max` của rule), `issue`, `fix`, `context`

**Rule ID discipline (BẮT BUỘC):**
- **Chỉ dùng 21 canonical rule IDs** đã list ở [SKILL.md Step 4](../SKILL.md#step-4--apply-rules). KHÔNG tự bịa rule mới (`INSECURE-COOKIE`, `AUTH-BYPASS`, `WEAK-CRYPTO`, `DATA-IN-URL`, `OAUTH-MISCONFIG`, `SUPPLY-CHAIN`, `INFO-DISCLOSURE`, `DATA-AT-REST`...).
- Nếu phát hiện 1 issue mà không có rule khớp 100%, MAP về rule canonical gần nhất (xem mapping table trong [`../references/sub-agent-prompts.md`](../references/sub-agent-prompts.md#rule-id-discipline-critical--read-carefully)) và note lý do trong `issue`.
- 1 dòng code dính 2 rule (vd IDOR + RACE) → tạo **2 finding riêng biệt**, mỗi cái 1 `rule_id`. Không gom comma-separated.

### Step S3 — Cross-rule checks

Một số rule cần đối chiếu cross-file:

- **SLOPSQUATTING**: thu thập tất cả import statement → kiểm tra package name có phải hợp lệ (npm/PyPI/Packagist). Trong môi trường offline, dùng heuristics: tên có lookalike chars (l/I, 0/O), suffix `-js`/`-py` lạ, hoặc tên typo phổ biến (vd `requets` thay vì `requests`).
- **OUTDATED-DEPENDENCY**: đọc `package.json` / `requirements.txt` / `go.mod` / `composer.json` / `Gemfile.lock` → check version có trong list known-vulnerable (rule sẽ có danh sách static, không fetch internet).
- **IDOR + BROKEN-ACCESS-CONTROL**: cần đọc cả route definition + handler để verify có authz check.
- **MASS-ASSIGNMENT**: cần đọc model definition + endpoint handler.
- **CSRF**: kiểm tra middleware config global (Express `csurf`, Laravel `VerifyCsrfToken`...) trước khi flag từng endpoint.

### Step S4 — Build PASSED list

Với mỗi rule applicable đã check và **không phát hiện vấn đề**, add vào PASSED list với 1 dòng giải thích đã check gì.

Rule **không applicable** (vd: PHP rules trong repo thuần Node) → không đưa vào PASSED list, không hiện.

### Step S5 — Determine verdict

| Findings | Verdict |
|---|---|
| ≥1 CRITICAL | FAIL |
| 0 CRITICAL, ≥1 HIGH | WARN |
| 0 CRITICAL, 0 HIGH | PASS |

MEDIUM/LOW không ảnh hưởng verdict cuối nhưng vẫn render.

### Step S6 — Render report (v0.3+ verbose)

Theo template trong [`../references/output-format.md`](../references/output-format.md). Dùng i18n strings từ `$LANG` file.

**Render order:**
1. Header
2. Verdict + description
3. CRITICAL section: **overview table + verbose blocks per finding** (Mô tả ngắn + Tại sao nguy hiểm + Attack scenario + Code before/after + Đọc thêm)
4. HIGH section: **overview table + medium blocks per finding** (Mô tả + Tác động + Code fix + Đọc thêm)
5. MEDIUM section (compact table, 1 row/finding)
6. LOW section (compact table, 1 row/finding)
7. PASSED section
8. Next steps
9. **Save notification + gitignore warning**
10. Footer
11. JSON summary (canonical EN, fenced ```json)

**Per-finding data cần thu thập** (để render verbose):

Khi đang scan, với mỗi finding, ngoài `file`, `line`, `rule_id`, `severity`, `issue`, `fix`, agent cần thu thập:
- `short_desc` (1 dòng cho overview table)
- Nếu CRITICAL: `why_dangerous` (1 đoạn), `attack_scenario` (steps numbered), `code_before` (snippet code thật từ file), `code_after` (snippet đã sửa)
- Nếu HIGH: `impact` (1 đoạn), `fix_code` (snippet với before→after comment)
- Nếu MEDIUM/LOW: chỉ `short_desc`

Nguồn cho `why_dangerous` + `attack_scenario`: nội dung rule file (section "Intent" + "Search patterns" giải thích). Agent paraphrase cho non-tech user, KHÔNG copy nguyên block từ rule.

### Step S7 — Output + Save (v0.3+)

1. **Print toàn bộ report ra stdout** (Claude Code hiển thị cho user)
2. **Save same content to file** dùng Write tool:
   - Path: `vbsec-reports/scan-<TIMESTAMP>.md` (đã prepare ở SKILL.md Step 0)
   - Nội dung IDENTICAL với stdout
3. **Print save notification** ra stdout (sau report, trước JSON):
   ```
   📄 {msg_report_saved}: vbsec-reports/scan-2026-05-13-143022.md
   ```
4. **If gitignore warning needed** (check from SKILL.md Step 0):
   ```
   ⚠️ {msg_gitignore_warning_title}: {msg_gitignore_warning_text}
   ```

LƯU Ý: i18n key `msg_report_saved` và `msg_gitignore_warning_*` đã có trong `references/i18n/{vi,en}.md`.

## Tips để giảm context burn

- **Nạp rule 1 lần qua `load-rules.sh`** (chỉ phần phát hiện), giữ trong context xuyên suốt. Phần chi tiết chỉ đọc cho rule có finding CRITICAL/HIGH
- **Không read file 2 lần** trong cùng scan — nếu đã đọc 1 file vì rule X, dùng lại context khi check rule Y
- **Grep trước, Read sau**: tìm hot spots bằng Grep (rẻ), chỉ Read khi cần verify
- **Skip aggressive**: file đã đọc và không có pattern khả nghi → mark "scanned" và move on
- **Batch tools**: Grep 1 lần cho nhiều patterns nếu tool hỗ trợ (vd: regex alternation)

## Performance target

Repo SMALL điển hình (10-30 files):
- Tool calls: 30-80 (Grep + Read combined)
- Time: 30-60 giây
- Context burn: ~30-50K tokens

Nếu vượt 100 tool calls hoặc 80K tokens → review có nên switch sang LARGE mode hay không (main agent có thể tự re-route).

## Quick verification

Sau khi render report, double-check:
- [ ] Mọi finding có file path + line number cụ thể
- [ ] CRITICAL severity dùng đúng cho L1 → dangerous sink, no sanitization
- [ ] JSON summary cuối có đầy đủ fields theo schema
- [ ] PASSED list có ít nhất các rule applicable đã check (không bỏ sót)
- [ ] Không có chuỗi tiếng Việt/Anh hardcoded — đã dùng i18n key
- [ ] **Mọi `rule_id` trong findings nằm trong 21 canonical IDs** (không có bịa)
- [ ] **Counts sanity check**: `len(findings)` == `summary.critical + high + medium + low`
- [ ] Nếu render `top_rules_by_count`: tổng count == `len(findings)` (không double-count)
