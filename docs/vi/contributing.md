# Đóng góp cho vbsec

Cảm ơn bạn quan tâm tới vbsec! Tài liệu này hướng dẫn workflow để thêm rule mới, language specialization mới, hoặc cải thiện rule existing.

> Đối tượng: developer biết git + cơ bản về security. Bạn KHÔNG cần biết Claude Code SDK — skill chỉ là markdown.

---

## Mục lục

- [Setup dev environment](#setup-dev-environment)
- [Loại đóng góp được welcome](#loại-đóng-góp-được-welcome)
- [Workflow chung](#workflow-chung)
- [Thêm rule generic mới](#thêm-rule-generic-mới)
- [Thêm language specialization](#thêm-language-specialization)
- [Thêm i18n string](#thêm-i18n-string)
- [Testing trước khi PR](#testing-trước-khi-pr)
- [Coding style cho rule file](#coding-style-cho-rule-file)
- [PR checklist + commit conventions](#pr-checklist--commit-conventions)

---

## Setup dev environment

```bash
# Fork repo trên GitHub trước, rồi:
git clone git@github.com:<your-username>/vbsec.git ~/vbsec-dev
cd ~/vbsec-dev
git remote add upstream https://github.com/tanviet12/vbsec.git

# Symlink folder skill vào Claude Code để edit là test ngay
mkdir -p ~/.claude/skills
ln -sfn ~/vbsec-dev/skill ~/.claude/skills/vbs-scan-security

# Restart Claude Code
```

Edit bất kỳ file nào trong `skills/vbs-scan-security/` → mở Claude Code → gọi `/vbs-scan-security` → file mới đã được load (Claude Code đọc file fresh mỗi lần invoke skill).

**Mẹo:** Tạo 1 test repo riêng để dogfood:

```bash
mkdir ~/vbsec-test && cd ~/vbsec-test
git init
# Bỏ vào đây các file unsafe để test rule
```

---

## Loại đóng góp được welcome

| Loại | Độ ưu tiên | File chạm tới |
|---|---|---|
| Thêm language specialization (Ruby, JS/TS, Python, Java, Rust) | Cao | `skills/vbs-scan-security/rules/languages/<lang>/` |
| Cải thiện rule existing (thêm pattern, giảm false positive) | Cao | `skills/vbs-scan-security/rules/generic/NN-*.md` |
| Sửa lỗi reasoning trong rule | Cao | `skills/vbs-scan-security/rules/generic/NN-*.md` |
| Thêm test case (positive + negative) | Trung | `tests/fixtures/<bộ>/` + `tests/expected/<bộ>.json` (xem [`tests/README.md`](../../tests/README.md)) |
| Thêm rule mới (rule 32, 33...) | Trung | Cần thảo luận trước qua issue |
| Cập nhật CVE list cho OUTDATED-DEPENDENCY (fallback offline) | Trung | `skills/vbs-scan-security/rules/generic/20-outdated-dependency.md` |
| Thêm ecosystem mới cho `--sca`/VULNERABLE-DEPENDENCY | Trung | `skills/vbs-scan-security/references/dependency-scan.md` |
| Docs (typo, ví dụ, cải thiện workflow) | Thấp-Trung | `docs/`, `README.*.md` |

**Trước khi làm PR lớn (rule mới, language mới):** mở issue thảo luận để tránh dụng độ.

---

## Workflow chung

```
1. Fork + clone + symlink
       ↓
2. Branch: feat/add-ruby-specialization
       ↓
3. Edit file rule trong skills/vbs-scan-security/
       ↓
4. Test trên test repo (positive + negative case)
       ↓
5. Cập nhật docs/vi/rules.md + docs/en/rules.md
       ↓
6. Cập nhật i18n nếu có string mới
       ↓
7. Commit theo convention (feat/fix/docs/lang)
       ↓
8. Push, open PR, mô tả test plan
```

---

## Thêm rule generic mới

### 1. Đặt tên + chọn số

Rule kế tiếp là **32** (01–21 kinh điển, 22 là `VULNERABLE-DEPENDENCY` cho `--sca`, 23–31 là rule SaaS / agent AI thêm ở v0.8). Tên ID dạng `KEBAB-CASE-UPPERCASE`, ví dụ: `OPEN-REDIRECT`, `XML-XXE`, `LDAP-INJECTION`.

Lỗi chưa có rule vẫn được báo với mã `OTHER-CWE-<n>`. Chỉ nên thêm rule mới khi một loại lỗi lặp lại nhiều và cần cách suy luận riêng (khi nào tính, khi nào không, mức độ).

File: `skills/vbs-scan-security/rules/generic/32-open-redirect.md`

### 2. Template frontmatter

```markdown
---
id: OPEN-REDIRECT
severity_max: HIGH
applies_to: all
cwe: [CWE-601]
owasp: [A01:2021]
# load_when: 'regex'   # tuỳ chọn: chỉ nạp khi repo khớp (xem rule 23–25)
---

# OPEN-REDIRECT — Tên dễ hiểu của lỗ hổng

## Intent

(1 đoạn ngắn: vì sao rule này quan trọng, scenario thực tế khi bị khai thác)

## When CRITICAL / HIGH / MEDIUM

| Điều kiện | Severity |
|---|---|
| <điều kiện 1> | CRITICAL |
| <điều kiện 2> | HIGH |
| <điều kiện 3> | MEDIUM |

## Reasoning

(Hướng dẫn LLM agent cách quyết định: trace L1-L4, đọc context, etc.)

## Search patterns (illustrative — KHÔNG phải lệnh chạy literal)

```bash
# Ví dụ pattern để Grep
grep -rE 'res\.redirect\(req\.(query|body|params)' src/
```

## Examples

### Unsafe
```<lang>
<code>
```

### Safe
```<lang>
<code>
```

## Fix recommendation

(Đoạn ngắn về cách sửa, để vbsec đưa vào cột "Cách sửa" trong báo cáo)

## Cross-references

- OWASP: <link>
- CWE: CWE-601 (Open Redirect)
- Related rules: SSRF, CSRF
```

### 3. Update các file phụ thuộc

| File | Cần làm gì |
|---|---|
| [`skills/vbs-scan-security/SKILL.md`](../../skills/vbs-scan-security/SKILL.md) | Thêm row vào bảng rule ở Step 4 ("30 rule generic" → "31") |
| [`workflows/auto-fix.md`](../../skills/vbs-scan-security/workflows/auto-fix.md) | Xếp rule vào nhóm sửa cơ học (tự áp) hoặc nhóm quyền / logic (`suggested_only`) |
| [`docs/vi/rules.md`](rules.md) | Thêm section `### Rule 32 — OPEN-REDIRECT` |
| [`docs/en/rules.md`](../en/rules.md) | Thêm tương ứng |
| [`README.vi.md`](../../README.vi.md) | Thêm ID vào bảng "vbsec bắt được những lỗi nào", sửa số rule |
| [`README.md`](../../README.md) | Tương ứng ("What vbsec catches") |
| `tests/expected/<bộ>.json` | Ít nhất một `must_find` và một bẫy an toàn cho rule mới |

Xong chạy `./scripts/sync-skills.sh` để chép rule sang bản Codex và Antigravity.

### 4. Test

Xem [Testing trước khi PR](#testing-trước-khi-pr).

---

## Thêm language specialization

Specialization = override rule generic cho 1 ngôn ngữ cụ thể, với pattern + reasoning chính xác hơn cho framework/library phổ biến của lang đó.

### Ví dụ: thêm Ruby

```bash
mkdir -p skills/vbs-scan-security/rules/languages/ruby
```

Chọn rule muốn specialize (không cần toàn bộ 30 — chọn rule nào Ruby có pattern đặc thù). Ví dụ:

- SQL-INJECTION → ActiveRecord `.where("...")` raw vs parameterized
- MASS-ASSIGNMENT → `User.update(params)` thiếu `permit`
- INSECURE-DESERIALIZATION → `Marshal.load`, `YAML.load`

### Template specialization file

File: `skills/vbs-scan-security/rules/languages/ruby/02-sql-injection.md` (số + tên trùng với generic counterpart):

```markdown
---
id: SQL-INJECTION
severity_max: CRITICAL
applies_to: ruby
overrides: generic/02-sql-injection.md
---

# SQL-INJECTION — Ruby/Rails specialization

## Ruby-specific patterns

### Pattern 1: ActiveRecord raw where
```ruby
# UNSAFE
User.where("name = '#{params[:name]}'")

# SAFE
User.where("name = ?", params[:name])
User.where(name: params[:name])
```

### Pattern 2: find_by_sql
```ruby
# UNSAFE
User.find_by_sql("SELECT * FROM users WHERE email = '#{email}'")

# SAFE
User.find_by_sql(["SELECT * FROM users WHERE email = ?", email])
```

## Search patterns (illustrative)

```bash
grep -rE '\.where\("[^"]*#\{' app/
grep -rE 'find_by_sql.*#\{' app/
```

## Reasoning

(Override / extend reasoning của generic rule cho Ruby idioms)
```

### Update các file phụ thuộc

| File | Cần làm gì |
|---|---|
| [`skills/vbs-scan-security/references/language-detection.md`](../../skills/vbs-scan-security/references/language-detection.md) | Thêm Ruby vào Phase 1 table (extension `.rb`, file `Gemfile`) |
| [`skills/vbs-scan-security/SKILL.md`](../../skills/vbs-scan-security/SKILL.md) | Dòng "Phase 1 hiện hỗ trợ chuyên sâu: `go`, `php`" → thêm `ruby` |
| [`docs/vi/rules.md`](rules.md) | Cập nhật cột "Specialization" cho rule có override Ruby |
| [`docs/en/rules.md`](../en/rules.md) | Tương ứng |
| [`README.vi.md` + `README.md`](../../README.vi.md) | Update Roadmap (đánh dấu Ruby done) |

### Folder README

Có thể thêm `skills/vbs-scan-security/rules/languages/ruby/README.md` nói qua: framework cover được (Rails, Sinatra), ORM (ActiveRecord), conventions Ruby idiomatic.

---

## Thêm i18n string

Mọi user-facing string trong báo cáo phải qua i18n. Khi cần thêm key mới:

1. Mở `skills/vbs-scan-security/references/i18n/vi.md` — thêm key + Vietnamese value
2. Mở `skills/vbs-scan-security/references/i18n/en.md` — thêm CÙNG key + English value
3. Trong file rule / template, reference key (không hardcode string)

**Quy tắc:**
- Key dùng `snake_case` (`fix_recommendation`, `verdict_fail`)
- KHÔNG dịch: rule ID, file path, code snippet, command name
- 2 file vi/en phải cùng số key — thiếu 1 bên là bug

---

## Testing trước khi PR

### Bộ fixture

Repo có sẵn các app mẫu cài lỗi kèm đáp án (chi tiết ở [`tests/README.md`](../../tests/README.md)):

- Bộ kinh điển: `go`, `php`, `typescript`, `python`, `dotnet`, `dart`, cùng bản khó `go-hard`, `php-hard`, `typescript-hard`.
- Bộ SaaS (v0.8): `saas-hard` (máy chủ MCP, agent quá quyền, tenant, phân quyền lệch giữa các đường, lộ dữ liệu), `ledger-hard` (luật tiền và trạng thái, đồng thời, log, phiên), `misc-hard` (bom giải nén, log chứa dữ liệu cá nhân, open redirect), `lessons-hard` (mỗi bài học bảo mật một ca). Mỗi bộ có file luật dự án mẫu ở `tests/context/<bộ>.md` để truyền bằng `--context`.

```bash
./scripts/run-fixtures.sh saas-hard      # quét thật + chấm điểm (tốn token)
./scripts/run-fixtures.sh --check-only   # chỉ chấm báo cáo mới nhất đã có
```

### Bài học → ca kiểm

`scripts/check-lessons.py` kiểm mỗi bài học bảo mật trong kho bài học đều có ca kiểm. Kho bài học là file riêng của từng người bảo trì (mặc định `~/.claude/rules/bai-hoc-bao-mat.md`, đổi bằng `--lessons` hoặc `VBSEC_LESSONS`); repo chỉ giữ bảng nối `tests/lessons-map.json`. Mỗi bài ở một trong ba trạng thái: `covered` (ca phải có trong `tests/expected/<bộ>.json`), `out_of_scope` (kèm `reason`), `todo`. Script báo đỏ khi có bài chưa nối, bài `todo`, ca trỏ tới đáp án không tồn tại, hoặc tiêu đề bài đã đổi. Không có kho bài học → cảnh báo, exit 0.

```bash
python3 scripts/check-lessons.py
python3 scripts/check-lessons.py --init   # in khung JSON cho bài chưa nối
```

### Thử tay nhanh

Sửa một rule đơn lẻ thì thường hai ca nhỏ là đủ:

### Positive case — rule PHẢI flag

```bash
mkdir ~/vbsec-test-positive && cd ~/vbsec-test-positive
git init

# Tạo file unsafe cho rule mới
cat > app.py <<'EOF'
import os
filename = request.args.get('file')
os.system(f"convert {filename} output.png")  # command injection
EOF

git add app.py

# Test
# Trong Claude Code session ở thư mục này:
#   /vbs-scan-security staged
# Báo cáo PHẢI có CRITICAL với rule COMMAND-INJECTION ở app.py:3
```

### Negative case — rule KHÔNG được false-positive

```bash
mkdir ~/vbsec-test-negative && cd ~/vbsec-test-negative
git init

# Tạo file safe cho rule (cách viết đúng)
cat > app.py <<'EOF'
import subprocess
subprocess.run(['convert', filename, 'output.png'], check=True)
EOF

git add app.py

# /vbs-scan-security staged
# Báo cáo PHẢI có COMMAND-INJECTION trong "ĐÃ ĐẠT" (không có CRITICAL/HIGH)
```

### Edge case — pattern legitimate

Nhiều pattern trông giống vuln nhưng safe trong context cụ thể. Ví dụ: `f"SELECT * FROM users WHERE id={user_id}"` SAFE nếu `user_id` là biến internal L3 (hardcoded const, framework-provided). Test đảm bảo rule KHÔNG flag những case này.

### Test `--auto-fix` / `--sca`

2 flag này mutate file / gọi network — test rõ cả nhánh fail, không chỉ happy path:

- **`--auto-fix`:** chạy trên repo có finding CRITICAL cố tình **không sửa được** (vd fix đúng cần 1 dependency chưa install, nên build luôn fail). Xác nhận: file được revert về nguyên bản sau khi hết retry budget, `patch_status: "failed_verification"`, và working tree sạch (`git diff` rỗng cho file đó). Lặp lại khi file đó đang có thay đổi chưa commit (`uncommitted --auto-fix`) và xác nhận thay đổi đó vẫn còn sau khi chạy. Chạy thêm 1 lần khi không có build tool trong `PATH` (vd không có `dotnet`) và xác nhận không file nào bị sửa (`suggested_only`).
- **`--sca`:** chạy 1 lần có network (xác nhận `scan_source: "osv.dev live"`) và 1 lần block network, vd kết hợp `--sca` với DNS/proxy không reachable (xác nhận fallback graceful: `scan_source: "static list (offline)"`, không crash, có in `{msg_sca_unavailable}`).

### Document test plan trong PR

Trong mô tả PR, bao gồm:

```markdown
## Test plan

- [x] Positive: app/sql_injection_unsafe.rb → flagged CRITICAL SQL-INJECTION
- [x] Negative: app/sql_injection_safe.rb → no finding
- [x] Edge: app/sql_injection_internal_const.rb → no finding (L3 input)
```

---

## Coding style cho rule file

- **Ngôn ngữ prose:** Tiếng Việt cho diễn giải, English cho rule ID + code identifier
- **Code snippet:** giữ language tag chính xác (` ```python `, ` ```javascript `, ` ```go `, ` ```php `)
- **KHÔNG dùng emoji** trừ checkmark `✓` / `✗` trong bảng (giữ format clean cho machine parse)
- **Length:** mỗi rule file 100-300 dòng. Ngắn quá thiếu context, dài quá LLM context-bloat.
- **Pattern bash trong rule là MINH HỌA:** ghi rõ "illustrative" để LLM agent biết KHÔNG chạy literal. Skill dùng Grep/Read tool, không Bash grep.
- **Cross-reference:** mỗi rule có link OWASP + CWE + related rules

### Heading hierarchy chuẩn

```markdown
# RULE-ID — Tên dễ hiểu

## Intent
## When CRITICAL / HIGH / MEDIUM
## Reasoning
## Search patterns (illustrative)
## Examples
### Unsafe
### Safe
## Fix recommendation
## Cross-references
```

---

## PR checklist + commit conventions

### Checklist trước khi mở PR

- [ ] Branch tách từ `main` mới nhất
- [ ] Rule file có frontmatter đủ (id, severity_max, applies_to, cwe, owasp; load_when nếu chỉ nạp ở một số repo)
- [ ] Đã test positive case → flag đúng
- [ ] Đã test negative case → KHÔNG false-positive
- [ ] Đã chạy `./scripts/run-fixtures.sh <bộ>` cho các bộ thay đổi đụng tới
- [ ] `python3 scripts/check-lessons.py` xanh (nếu có kho bài học)
- [ ] Đã chạy `./scripts/sync-skills.sh`
- [ ] Đã update `docs/vi/rules.md` + `docs/en/rules.md`
- [ ] Đã update `SKILL.md` Step 4 table nếu rule mới
- [ ] Đã update README.vi.md + README.md nếu rule mới
- [ ] Nếu thêm string mới: cập nhật cả `i18n/vi.md` VÀ `i18n/en.md`
- [ ] Commit message theo convention bên dưới
- [ ] PR description có test plan

### Commit conventions

| Prefix | Khi nào dùng | Ví dụ |
|---|---|---|
| `feat:` | Thêm tính năng / rule mới | `feat: add OPEN-REDIRECT rule (32)` |
| `fix:` | Sửa bug, giảm false positive | `fix: reduce false positive in SQL-INJECTION for parameterized GORM` |
| `docs:` | Cập nhật docs / README | `docs: clarify SMALL vs LARGE thresholds` |
| `lang:` | Thêm/sửa language specialization | `lang: add Ruby specialization for SQL-INJECTION` |
| `refactor:` | Restructure rule file, không đổi behavior | `refactor: split JWT rule into 3 sub-checks` |
| `chore:` | Build, CI, formatting | `chore: add markdownlint config` |

Format đầy đủ: `<type>: <imperative summary>` (≤72 ký tự), xuống dòng + body giải thích why nếu cần.

### PR title

Theo cùng convention: `feat: add Ruby specialization`. PR description nên có:

```markdown
## What

(1-2 câu mô tả ngắn)

## Why

(Tại sao cần thay đổi này)

## Test plan

- [ ] Positive case ...
- [ ] Negative case ...

## Checklist

- [ ] docs/vi/rules.md updated
- [ ] docs/en/rules.md updated
- [ ] i18n updated (nếu có)
```

---

## Câu hỏi thường gặp

**Q: Có cần biết Claude Code SDK không?**
A: Không. Skill là pure markdown. Bạn chỉ cần biết viết .md + hiểu intent của rule.

**Q: Làm sao test mà không tốn token API?**
A: Tạo test repo nhỏ (1-3 file), dùng SMALL mode (~30s mỗi lần scan). Avoid `all` scope khi đang dev.

**Q: Rule pattern bash có chạy được không?**
A: Đó là minh họa cho LLM hiểu, KHÔNG phải lệnh execute. Skill dùng Grep tool của Claude Code.

**Q: Có thể đóng góp rule mới mà không add vào SKILL.md table không?**
A: Không. Bảng đó là source of truth — Claude Code đọc SKILL.md để biết apply rule nào.

**Q: Còn câu hỏi khác?**
A: Mở [issue](https://github.com/tanviet12/vbsec/issues) hoặc discussion trên GitHub.
