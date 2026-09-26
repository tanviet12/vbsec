# Sử dụng vbsec

Hướng dẫn chi tiết cách gọi `/vbs-scan-security`, chọn scope, hiểu báo cáo, và tích hợp vào CI/CD.

> Đã cài skill chưa? Nếu chưa, đọc [installation.md](installation.md) trước.

---

## Mục lục

- [Lệnh cơ bản](#lệnh-cơ-bản)
- [Tất cả scope](#tất-cả-scope)
- [Báo cáo lưu ở đâu](#báo-cáo-lưu-ở-đâu)
- [Chọn ngôn ngữ output](#chọn-ngôn-ngữ-output)
- [Anatomy of a report](#anatomy-of-a-report)
- [JSON summary cho tooling](#json-summary-cho-tooling)
- [Performance expectations](#performance-expectations)
- [Resume khi LARGE mode bị interrupt](#resume-khi-large-mode-bị-interrupt)
- [Auto-fix (--auto-fix)](#auto-fix---auto-fix)
- [Quét dependency live (--sca)](#quét-dependency-live---sca)
- [Tích hợp CI/CD](#tích-hợp-cicd)

---

## Lệnh cơ bản

Mở Claude Code ở thư mục git repo. Trong khung chat:

```
/vbs-scan-security
```

> **⚠️ Thay đổi hành vi ở v0.3:** `/vbs-scan-security` (không kèm tham số) bây giờ quét **toàn bộ repository**. Trước đây mặc định tương đương với `uncommitted`. Nếu muốn hành vi cũ, hãy dùng `/vbs-scan-security uncommitted` hoặc `/vbs-scan-security diff`.

Mặc định vbsec sẽ:
1. Lấy tất cả file trong repo (toàn bộ, không chỉ uncommitted)
2. Detect ngôn ngữ chính (Go/PHP/JS/Python/...)
3. Route SMALL hoặc LARGE mode theo số file
4. Áp 21 rule generic + overlay language (nếu có)
5. In báo cáo Markdown + JSON summary lên stdout
6. Lưu một bản báo cáo vào `vbsec-reports/scan-<timestamp>.md` trong repo

---

## Tất cả scope

| Lệnh | Quét gì |
|---|---|
| `/vbs-scan-security` | **Toàn bộ repo (mặc định — thay đổi ở v0.3)** |
| `/vbs-scan-security uncommitted` | Chỉ thay đổi chưa commit (staged + unstaged) |
| `/vbs-scan-security diff` | Alias của `uncommitted` |
| `/vbs-scan-security staged` | Chỉ file đã `git add` |
| `/vbs-scan-security commit within 7days` | Tất cả file đụng tới trong N ngày qua |
| `/vbs-scan-security commit id <sha>` | File trong 1 commit cụ thể |
| `/vbs-scan-security pr id 42` | File trong PR #42 (GitHub) — cần `gh` CLI |
| `/vbs-scan-security all` | Alias rõ ràng cho toàn bộ repo |

### Ví dụ thực tế

```bash
# Mặc định — quét toàn bộ repo, báo cáo Tiếng Việt, lưu vào file
/vbs-scan-security

# Trước khi commit: chỉ quét những gì đã thay đổi
/vbs-scan-security uncommitted

# Review trước khi merge: scan PR với báo cáo tiếng Anh
/vbs-scan-security pr id 42 lang=en

# Audit định kỳ: scan commit gần đây
/vbs-scan-security commit within 30days

# Tìm báo cáo mới nhất
ls -lt vbsec-reports/ | head -5

# So sánh 2 lần scan
diff vbsec-reports/scan-2026-05-13-100000.md vbsec-reports/scan-2026-05-14-100000.md
```

---

## Báo cáo lưu ở đâu

Từ v0.3 trở đi, **mỗi lần scan vbsec đều lưu một bản báo cáo vào file** ngay trong repo được quét:

- Đường dẫn: `vbsec-reports/scan-<YYYY-MM-DD-HHMMSS>.md`
- Định dạng tên file: `scan-YYYY-MM-DD-HHMMSS.md` — sort theo thứ tự thời gian tự nhiên
- Nội dung file giống hệt nội dung in ra stdout — file dùng để đọc lại và share, stdout để xem ngay
- Nếu `vbsec-reports/` chưa có trong `.gitignore`, vbsec sẽ in một khuyến nghị ở cuối stdout — **vbsec không tự sửa `.gitignore`**, việc đó để bạn quyết định
- Báo cáo có thể xóa bất cứ lúc nào, không phải file state

Tìm báo cáo mới nhất:

```bash
ls -lt vbsec-reports/ | head -5
```

Xem báo cáo gần nhất:

```bash
cat "$(ls -t vbsec-reports/scan-*.md | head -1)"
```

Xem [installation.md](installation.md#sau-khi-cài-nơi-lưu-báo-cáo-và-gitignore) để thêm `vbsec-reports/` vào `.gitignore`.

---

## Chọn ngôn ngữ output

vbsec hỗ trợ Tiếng Việt (mặc định) và English. Cách chỉ định:

| Cú pháp | Kết quả |
|---|---|
| (không có flag) | Tiếng Việt |
| `lang=vi` | Tiếng Việt |
| `--vi` | Tiếng Việt |
| `lang=en` | English |
| `--en` | English |

Ví dụ:

```
/vbs-scan-security pr id 42 lang=en
/vbs-scan-security commit within 14days --en
/vbs-scan-security staged
```

**Lưu ý:** Báo cáo Markdown đổi theo lang, NHƯNG:
- Rule ID (HARDCODED-SECRET, SQL-INJECTION...) luôn EN
- File path, code snippet luôn nguyên gốc
- JSON summary ở cuối báo cáo LUÔN dùng EN canonical (để tooling parse ổn định)

---

## Anatomy of a report

Một báo cáo điển hình:

```
# Báo cáo quét bảo mật vbsec

Phạm vi: Thay đổi chưa commit
Số file: 12 (8 .ts, 4 .py)
Ngôn ngữ chính: typescript
Chế độ: NHỎ (quét trực tiếp)
Ngày: 2026-05-13

## KẾT LUẬN: KHÔNG ĐẠT
Có lỗi NGHIÊM TRỌNG. KHÔNG được deploy đến khi sửa hết.

## NGHIÊM TRỌNG (chặn deploy)
| File:Dòng | Loại lỗi | Mô tả | Cách sửa |
|---|---|---|---|
| api/users.ts:42 | SQL-INJECTION | req.body.id ghép vào SQL | Dùng parameterized query |
| .env:1 | HARDCODED-SECRET | STRIPE_KEY=sk_live_... | Xoay key + .gitignore |

## CAO (cần khắc phục)
| auth.ts:18 | WEAK-PASSWORD-HASHING | createHash('md5') | Dùng bcrypt/argon2 |

## TRUNG BÌNH
(không có)

## ĐÃ ĐẠT
- ✓ XSS — Vue auto-escape
- ✓ CORS — Origin cụ thể
- ✓ CSRF — Token có trong form

## JSON Summary
```json
{
  "verdict": "FAIL",
  "summary": {"critical": 2, "high": 1, "medium": 0, "low": 0, "passed": 14},
  "scope": "uncommitted",
  "files_reviewed": 12,
  "primary_language": "typescript",
  "specialized_rules_used": true,
  "mode": "small",
  "date": "2026-05-13",
  "findings": [
    {"file": "api/users.ts", "line": 42, "rule_id": "SQL-INJECTION", "severity": "CRITICAL", "issue_summary": "req.body.id ghép thẳng vào SQL", "fix_summary": "Dùng parameterized query"},
    {"file": ".env", "line": 1, "rule_id": "HARDCODED-SECRET", "severity": "CRITICAL", "issue_summary": "Stripe live key đã commit", "fix_summary": "Xoay key + .gitignore"},
    {"file": "auth.ts", "line": 18, "rule_id": "WEAK-PASSWORD-HASHING", "severity": "HIGH", "issue_summary": "MD5 dùng để hash password", "fix_summary": "Dùng bcrypt/argon2"}
  ]
}
```

### Các phần trong báo cáo

| Section | Mục đích |
|---|---|
| Header | Scope đã quét, số file, ngôn ngữ chính, mode (SMALL/LARGE), ngày |
| KẾT LUẬN (Verdict) | PASS / WARN / FAIL — quyết định có deploy được không |
| NGHIÊM TRỌNG | CRITICAL — chặn deploy, fix trước tiên |
| CAO | HIGH — phải fix trước production |
| TRUNG BÌNH | MEDIUM — fix sớm, không block |
| ĐÃ ĐẠT | Rule đã check và pass — minh chứng đã quét đủ |
| JSON Summary | Machine-readable, để CI parse |

### Verdict logic

| Tình huống | Verdict | Ý nghĩa |
|---|---|---|
| ≥1 CRITICAL | **FAIL** | Không được deploy |
| 0 CRITICAL, ≥1 HIGH | **WARN** | Có thể deploy nhưng phải có plan fix |
| 0 CRITICAL, 0 HIGH | **PASS** | OK deploy |

WARN **không** có nghĩa là "approve". Đội security/tech lead vẫn phải review HIGH issues.

---

## JSON summary cho tooling

JSON summary luôn nằm ở cuối báo cáo, trong fenced code block `json`. Schema cố định (xem đầy đủ ở [`output-format.md`](../../skills/vbs-scan-security/references/output-format.md)):

```json
{
  "verdict": "PASS" | "WARN" | "FAIL",
  "summary": {"critical": <int>, "high": <int>, "medium": <int>, "low": <int>, "passed": <int>},
  "scope": "uncommitted" | "staged" | "commit_within_Ndays" | "commit_id_<sha>" | "pr_<num>" | "all",
  "files_reviewed": <int>,
  "primary_language": <string>,
  "specialized_rules_used": <bool>,
  "mode": "small" | "large",
  "date": <ISO 8601 date>,
  "findings": [
    {
      "file": <string>,
      "line": <int>,
      "rule_id": <string>,
      "severity": "CRITICAL" | "HIGH" | "MEDIUM" | "LOW",
      "issue_summary": <string>,
      "fix_summary": <string>
    }
  ],
  "dependencies_scanned": { "total": <int>, "ecosystems": [<string>], "vulnerable_count": <int>, "scan_source": "osv.dev live" | "static list (offline)" }
}
```

`dependencies_scanned` chỉ có khi chạy `--sca`. Finding có `rule_id: VULNERABLE-DEPENDENCY` có thêm `cve_id`/`osv_id`/`package`/`ecosystem`/`installed_version`/`fixed_version`/`severity_source`, và `cvss_vector` (nguyên chuỗi CVSS vector từ OSV, để tham khảo) nếu có. Finding đã qua `--auto-fix` có thêm `patch_status`.

### Parse JSON từ output

Báo cáo là Markdown — extract JSON bằng:

```bash
# Output đã save vào report.md
awk '/^```json$/,/^```$/' report.md | sed '1d;$d' > summary.json
jq '.verdict' summary.json
# "FAIL"
```

Hoặc trong Node.js:

```javascript
const fs = require('fs');
const md = fs.readFileSync('report.md', 'utf8');
const match = md.match(/```json\n([\s\S]*?)\n```/);
const summary = JSON.parse(match[1]);
if (summary.verdict === 'FAIL') process.exit(1);
```

---

## Performance expectations

| Mode | Trigger | Thời gian | Tài nguyên |
|---|---|---|---|
| SMALL | ≤20 file ngôn ngữ chính VÀ ≤30 file tổng VÀ ≤14 ngày | 30-60 giây | 1 agent, ~5-15 tool calls |
| LARGE | Vượt 1 trong 3 ngưỡng trên | 5-15 phút | Spawn sub-agents song song, mỗi chunk 1 agent |

LARGE mode dùng [chunking-strategy.md](../../skills/vbs-scan-security/references/chunking-strategy.md) để chia file theo top-level folder, mỗi sub-agent xử lý 1 chunk. Main agent aggregate findings + translate cuối.

### Khi nào dùng SMALL vs LARGE?

Bạn không cần lo — SKILL.md route tự động. Nhưng nếu muốn ép:

- Scope hẹp hơn (`staged`, `commit id`) → thường SMALL
- Scope rộng (`all`, `commit within 30days`) → LARGE
- Repo nhỏ (<30 file) → luôn SMALL kể cả `all`

---

## Resume khi LARGE mode bị interrupt

LARGE mode dùng `TodoWrite` để track từng chunk. Nếu bạn interrupt giữa chừng (Ctrl+C, sập máy, etc.):

1. Mở lại Claude Code trong cùng thư mục
2. Gọi lại lệnh ban đầu (cùng scope, cùng lang)
3. Skill sẽ thấy thư mục `.vbsec-tmp/` còn findings cũ → tiếp tục từ chunk chưa xong

**Quan trọng:**
- ĐỪNG xóa `.vbsec-tmp/` thủ công nếu muốn resume
- Sau khi scan xong, vbsec tự cleanup `.vbsec-tmp/`
- Thêm `.vbsec-tmp/` vào `.gitignore` của repo nếu chưa có:
  ```
  .vbsec-tmp/
  ```

---

## Tích hợp CI/CD

vbsec là Claude Code skill — chạy trong agent CLI. Có 2 cách integrate:

### A. Pre-commit hook (local, mỗi dev tự chạy)

`.git/hooks/pre-commit`:

```bash
#!/usr/bin/env bash
# Yêu cầu Claude Code đã cài + plugin vbsec đã cài (/plugin install vbsec@vbsec)
set -e

REPORT=$(mktemp)
claude --no-stream -p '/vbs-scan-security staged lang=en' > "$REPORT"

VERDICT=$(awk '/^```json$/,/^```$/' "$REPORT" \
  | sed '1d;$d' | jq -r '.verdict' 2>/dev/null || echo "")

case "$VERDICT" in
  FAIL)
    echo "vbsec FAIL — see findings:"
    cat "$REPORT"
    exit 1
    ;;
  WARN)
    echo "vbsec WARN — HIGH issues found, commit allowed but please fix soon."
    ;;
  PASS)
    echo "vbsec PASS"
    ;;
  *)
    echo "vbsec: could not parse verdict, allowing commit"
    ;;
esac
```

**Lưu ý:** Pre-commit chỉ block khi `verdict=FAIL` (có CRITICAL). WARN chỉ cảnh báo.

### B. GitHub Actions (PR check)

`.github/workflows/vbsec.yml`:

```yaml
name: vbsec security scan
on:
  pull_request:
    branches: [main]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install Claude Code
        run: |
          curl -fsSL https://claude.ai/install.sh | bash
          echo "$HOME/.claude/bin" >> $GITHUB_PATH

      - name: Cài plugin vbsec
        run: |
          # Trong Claude Code session: cài marketplace + plugin
          # (Chạy qua `claude --no-stream -p` nếu CI image hỗ trợ,
          # hoặc dùng env var pre-install của Claude Code khi feature này phát hành.)
          mkdir -p ~/.claude/plugins
          # Cách 1: pre-warm bằng cách clone plugin vào cache
          git clone https://github.com/tanviet12/vbsec.git ~/.claude/plugins/cache/vbsec
          # Cách 2 (recommended khi Claude Code CI tooling chín hơn):
          #   claude plugin marketplace add tanviet12/vbsec
          #   claude plugin install vbsec@vbsec

      - name: Run vbsec on PR
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          claude --no-stream -p "/vbs-scan-security pr id ${{ github.event.pull_request.number }} lang=en" \
            > vbsec-report.md

      - name: Post comment on PR
        uses: marocchino/sticky-pull-request-comment@v2
        with:
          path: vbsec-report.md

      - name: Fail on CRITICAL
        run: |
          VERDICT=$(awk '/^```json$/,/^```$/' vbsec-report.md \
            | sed '1d;$d' | jq -r '.verdict')
          [ "$VERDICT" = "FAIL" ] && exit 1 || true
```

### C. Polices tùy chỉnh

Bạn có thể parse JSON summary và áp policy riêng:

```bash
# Chỉ block nếu có ≥3 HIGH issues
HIGH_COUNT=$(jq -r '.summary.high' summary.json)
if [ "$HIGH_COUNT" -ge 3 ]; then
  echo "Too many HIGH issues ($HIGH_COUNT). Blocking."
  exit 1
fi

# Hoặc: chỉ block 1 số rule cụ thể
CRITICAL_RULES=$(jq -r '.findings[] | select(.severity=="CRITICAL") | .rule_id' summary.json)
if echo "$CRITICAL_RULES" | grep -q "HARDCODED-SECRET"; then
  echo "Hardcoded secret detected — blocking deploy."
  exit 1
fi

# Block nếu có CVE CRITICAL confirmed qua OSV từ --sca
jq -e '[.findings[] | select(.rule_id=="VULNERABLE-DEPENDENCY" and .severity=="CRITICAL")] | length == 0' summary.json \
  || { echo "CVE CRITICAL confirmed qua OSV — blocking."; exit 1; }
```

---

## Auto-fix (`--auto-fix`)

**v0.7+, mặc định TẮT.** Sau khi scan, vbsec tự sinh patch (unified diff) cho từng finding CRITICAL/HIGH, apply thử qua `git apply`, chạy build command tương ứng ngôn ngữ (`dotnet build`, `go build ./...`, `tsc --noEmit`, `php -l`, `python -m py_compile`) để verify, và revert + thử lại (tối đa 2 lần) nếu build fail.

```
/vbs-scan-security uncommitted --auto-fix
```

**Trước khi dùng:**
- **Bắt buộc git repo** — patch được kiểm tra và apply qua `git apply`. Không có git → skill tự skip bước này, chỉ báo warning.
- **Revert an toàn** — trước mỗi patch, vbsec snapshot các file sắp bị ghi (gồm cả thay đổi chưa commit và file untracked). Build fail → khôi phục đúng bản trước patch, không dùng `git checkout`.
- **Kiểm tra trước khi apply** — thiếu build tool (vd không có `dotnet`) hoặc project vốn đã build lỗi → không apply gì, chỉ ghi gợi ý patch.
- **Dependency** — chỉ ghi manifest + lockfile (npm `--package-lock-only`, Composer `--no-install`, `go get`, `dotnet restore`), không chạy install script. Dependency Python chỉ có gợi ý patch, không chạy `pip install`.
- **Commit hoặc backup working tree trước** — auto-fix ghi đè trực tiếp file nguồn (dù có build verify, đây vẫn là thay đổi thật trên đĩa).
- Chỉ CRITICAL/HIGH được auto-fix; MEDIUM/LOW luôn để nguyên.
- Ngôn ngữ chưa có build command tin cậy (ngoài .NET/Go/TS/PHP/Python) → vbsec chỉ ghi gợi ý patch vào `vbsec-reports/patches/`, không tự ghi đè.

Kết quả từng finding được ghi vào field `patch_status` trong JSON summary (`applied`/`failed_verification`/`suggested_only`/`skipped_low_severity`) — CI có thể gate tiếp trên field này (vd fail nếu còn `failed_verification`).

---

## Quét dependency live (`--sca`)

**v0.7+, mặc định TẮT, cần network.** Thay vì chỉ dùng static list offline (rule `OUTDATED-DEPENDENCY`), `--sca` tra cứu **live** qua [OSV.dev](https://osv.dev) cho dependency manifest của 5 ecosystem: NuGet (.NET), Go, npm (TypeScript/JS), Composer (PHP), PyPI (Python).

```
/vbs-scan-security all --sca
```

Kết quả: finding `rule_id: VULNERABLE-DEPENDENCY` kèm `cve_id`, `fixed_version` và severity của chính advisory (cùng nguyên chuỗi `cvss_vector`) lấy từ OSV — không phải suy đoán. Nếu network không khả dụng, vbsec tự fallback về static list (rule 20), không fail scan.

Kết hợp cả 2 flag để tự động nâng version dependency có CVE và verify build:

```
/vbs-scan-security all --sca --auto-fix
```

---

## Bước tiếp theo

- Đọc [rules.md](rules.md) để hiểu chi tiết 21 rule chính + rule 22 (SCA)
- Muốn thêm rule mới hoặc support ngôn ngữ mới? Xem [contributing.md](contributing.md)
