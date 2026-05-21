---
name: vbs-scan-security
description: Use when scanning code for security vulnerabilities. Use when user says "scan security", "kiểm tra bảo mật", "security audit", "review security", or invokes `/vbs-scan-security`. For large scans (>20 main-language files OR >30 total OR >14 days) processes chunks sequentially. Outputs bilingual reports (vi/en).
license: MIT
---

# vbsec — Security Scanner cho Vibe Coders (Codex CLI variant)

Quét lỗ hổng bảo mật cho code do AI sinh ra (vibe code). Bộ skill này check 21 lỗi bảo mật phổ biến nhất của vibe code, kế thừa kiến trúc SMALL/LARGE mode, tổng quát hóa cross-language + chuyên sâu cho Go/PHP/Python/TypeScript/.NET.

> Public repo: https://github.com/tanviet12/vbsec
> License: MIT
> **Platform:** OpenAI Codex CLI. Phiên bản Claude Code spawn parallel sub-agents; phiên bản này dùng **sequential chunking** để giữ portability — chậm hơn ~3× nhưng output identical.

## Invocation

Trong Codex CLI:

```
/skills              # mở skill picker, chọn vbs-scan-security
$vbs-scan-security   # invoke trực tiếp qua $-prefix
```

Hoặc nói tự nhiên: *"scan security cho repo này"* / *"kiểm tra bảo mật"* — Codex tự match description.

| Argument | Scope | Mô tả |
|---|---|---|
| (không args) | **Toàn repo** | Mặc định — quét toàn bộ repo |
| `all` | Toàn repo | Alias explicit |
| `uncommitted` / `diff` | Uncommitted changes | Staged + unstaged |
| `staged` | Staged files only | Pre-commit scan |
| `commit within Xdays` | Recent commits | Quét commit X ngày gần đây |
| `commit id <sha>` | Specific commit | Quét 1 commit |
| `pr id <number>` | Pull request | Quét PR diff (cần `gh` CLI) |

**Lựa chọn ngôn ngữ output** (thêm vào bất kỳ scope nào):
- `lang=vi` hoặc `--vi` → Tiếng Việt (mặc định)
- `lang=en` hoặc `--en` → English

Ví dụ:
```
$vbs-scan-security pr id 42 lang=en
$vbs-scan-security staged --vi
$vbs-scan-security commit within 7days
```

---

## CRITICAL: Cách dùng skill này (cho LLM agent)

**Các pattern bash/grep trong rule files là VÍ DỤ minh họa, KHÔNG phải lệnh chạy literal.**

### Nguyên tắc

1. **Lý luận, không pattern-match thuần** — Hiểu intent bảo mật đằng sau mỗi check, không chỉ tìm chuỗi
2. **Dùng tool phù hợp** — file-read và grep tool của Codex (tên có thể là `read_file`, `grep`, `shell`) thay vì chạy bash grep/find thô
3. **Đọc context đầy đủ** — Khi gặp pattern, đọc hàm xung quanh để hiểu đây có thực sự là lỗ hổng không
4. **Phân loại trust level** — Một query có format chuỗi chỉ nguy hiểm nếu data ghép vào là **L1 (untrusted)**

### Phân loại nguồn dữ liệu (L1–L4)

| Level | Nguồn | Tin cậy | Ví dụ |
|---|---|---|---|
| L1 | Input người dùng | **KHÔNG tin** | `req.body`, `$_GET`, `request.params`, HTTP header, file upload |
| L2 | Database | Bán tin | Giá trị từ DB nhưng nguồn gốc là user input |
| L3 | Code nội bộ | Tin | Hardcoded strings, config keys, computed values |
| L4 | Hệ thống | Tin | Env vars, file paths nội bộ, framework constants |

**Key insight:** `f"SELECT ... {x}"` SAFE nếu `x` là L3+. CRITICAL nếu `x` là L1 không qua parameterization.

Tham khảo chi tiết: [`references/data-flow-classification.md`](references/data-flow-classification.md).

---

## Workflow

```
┌─────────────────────────────────────────────────────────────────────┐
│            vbsec SCAN WORKFLOW (Codex — Sequential)                  │
├─────────────────────────────────────────────────────────────────────┤
│  [Step 0] Parse args → scope + lang                                  │
│  [Step 1] Gather files (git)                                         │
│  [Step 2] Detect primary code language                               │
│  [Step 3] Route by size:                                             │
│           SMALL (≤20 main, ≤30 total, ≤14d) → inline                 │
│           LARGE (vượt ngưỡng)                → sequential chunking   │
│  [Step 4] Apply 21 rules (generic + lang overlay)                    │
│  [Step 5] Generate bilingual report + save to vbsec-reports/         │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Step 0: Parse Arguments

Dùng shell tool ĐÚNG MỘT LẦN cho step này.

```bash
ARGS="${ARGUMENTS:-$1}"

# 0) Detect git availability (KHÔNG bắt buộc có git — v0.5.1+)
IS_GIT_REPO=true
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || IS_GIT_REPO=false

# 1) Extract lang flag (default vi)
LANG="vi"
if echo "$ARGS" | grep -qE 'lang=en|--en|\ben\b'; then LANG="en"; fi
if echo "$ARGS" | grep -qE 'lang=vi|--vi'; then LANG="vi"; fi

# 2) Extract scope
SCOPE=$(echo "$ARGS" | sed -E 's/(lang=(vi|en)|--vi|--en)//g' | xargs)

# 3) Gather files
NO_GIT_NOTE=""
case "$SCOPE" in
  "staged"|"uncommitted"|"diff"|"commit within "*|"commit id "*|"pr id "*)
    if [ "$IS_GIT_REPO" = false ]; then
      echo "{msg_scope_needs_git}"
      exit 1
    fi
    case "$SCOPE" in
      "staged")             FILES=$(git diff --cached --name-only) ;;
      "uncommitted"|"diff") FILES=$(git diff --name-only HEAD); [ -z "$FILES" ] && FILES=$(git diff --cached --name-only) ;;
      "commit within "*)    DAYS=$(echo "$SCOPE" | grep -oE '[0-9]+'); FILES=$(git log --since="${DAYS} days ago" --name-only --pretty=format: | sort -u | grep -v '^$') ;;
      "commit id "*)        SHA=$(echo "$SCOPE" | sed 's/commit id //'); FILES=$(git diff-tree --no-commit-id --name-only -r "$SHA") ;;
      "pr id "*)            PR=$(echo "$SCOPE" | sed 's/pr id //'); FILES=$(gh pr diff "$PR" --name-only) ;;
    esac
    ;;
  "all"|"")
    if [ "$IS_GIT_REPO" = true ]; then
      FILES=$(git ls-files)
    else
      # Non-git folder — walk filesystem, exclude folder system + vendored
      FILES=$(find . -type f \
        -not -path '*/.git/*' \
        -not -path '*/.next/*' \
        -not -path '*/.nuxt/*' \
        -not -path '*/.venv/*' \
        -not -path '*/.idea/*' \
        -not -path '*/.vscode/*' \
        -not -path '*/node_modules/*' \
        -not -path '*/vendor/*' \
        -not -path '*/dist/*' \
        -not -path '*/build/*' \
        -not -path '*/target/*' \
        -not -path '*/__pycache__/*' \
        -not -path '*/vbsec-reports/*' \
        2>/dev/null | sed 's|^\./||')
      NO_GIT_NOTE="true"
    fi
    ;;
  *)
    echo "Unknown scope: $SCOPE"
    exit 1
    ;;
esac

# 4) Strip noise (double-protect)
FILES=$(echo "$FILES" | grep -vE '(^|/)(node_modules|vendor|dist|build|\.next|\.nuxt|target|\.venv|__pycache__|\.git|vbsec-reports)/' || true)

# 5) Prepare save location
TIMESTAMP=$(date +"%Y-%m-%d-%H%M%S")
REPORT_DIR="vbsec-reports"
REPORT_FILE="${REPORT_DIR}/scan-${TIMESTAMP}.md"
mkdir -p "${REPORT_DIR}"

# 6) Check .gitignore (chỉ relevant nếu là git repo)
GITIGNORE_WARNING=""
if [ "$IS_GIT_REPO" = true ]; then
  if [ -f .gitignore ]; then
    grep -qE '^vbsec-reports/?$' .gitignore || GITIGNORE_WARNING="missing"
  else
    GITIGNORE_WARNING="missing"
  fi
fi

echo "Scope: ${SCOPE:-all (default)}"
echo "Lang: $LANG"
echo "Git repo: $IS_GIT_REPO"
echo "Files: $(echo "$FILES" | wc -l)"
echo "Report file: $REPORT_FILE"
[ "$NO_GIT_NOTE" = "true" ] && echo "Note: non-git folder — scanning all files via find"
```

**Lưu ý (v0.5.1+):** Skill chạy được trên cả non-git folder. Default scope (`all`) dùng `find` thay `git ls-files`. Các scope dựa vào git (`staged`, `uncommitted`, `commit within`, `commit id`, `pr id`) BẮT BUỘC git — báo `msg_scope_needs_git` rồi exit. Nếu `NO_GIT_NOTE=true`, report header in `{msg_no_git_note}`.

---

## Step 1: Load i18n Strings

Đọc file i18n tương ứng với `$LANG`:
- `lang=vi` → đọc [`references/i18n/vi.md`](references/i18n/vi.md)
- `lang=en` → đọc [`references/i18n/en.md`](references/i18n/en.md)

File i18n chứa bảng key→text cho toàn bộ user-facing strings. Mọi text trong report final phải lấy từ i18n, KHÔNG hardcode.

**Strings KHÔNG bao giờ dịch:** rule ID, file path, code snippet, command name.

---

## Step 2: Detect Primary Code Language

Đọc [`references/language-detection.md`](references/language-detection.md). Tóm tắt:

1. Count extension trong file list: `.go`, `.py`, `.php`, `.js`, `.ts`, `.jsx`, `.tsx`, `.rb`, `.java`, `.rs`, `.cs`, `.csproj`, `.sln`
2. Primary lang = lang chiếm ≥30% tổng files
3. Có `rules/languages/<lang>/` → load overlay; không có → chỉ dùng generic
4. Multi-lang repo (Go backend + Vue frontend) → load cả 2 overlay

**Hiện hỗ trợ chuyên sâu:** `go`, `php`, `typescript` (gộp JS+TS), `python`, `dotnet`.

---

## Step 3: Route by Size

| Điều kiện | Ngưỡng | Mode |
|---|---|---|
| Files ngôn ngữ chính | ≤20 | SMALL |
| Files ngôn ngữ chính | >20 | **LARGE** |
| Tổng files | ≤30 | SMALL |
| Tổng files | >30 | **LARGE** |
| Timespan (scope `commit within`) | ≤14 ngày | SMALL |
| Timespan | >14 ngày | **LARGE** |

BẤT KỲ điều kiện nào sang LARGE → dùng LARGE mode.

- **SMALL mode:** Read [`workflows/small-review.md`](workflows/small-review.md) — inline scan
- **LARGE mode:** Read [`workflows/large-review-sequential.md`](workflows/large-review-sequential.md) — chunk + xử lý **tuần tự** (Codex hiện chưa có cơ chế spawn parallel sub-agents từ skill; sequential giữ logic chunking + aggregation nhưng chạy 1 chunk/lần)

---

## Step 4: Apply Rules

Cho mỗi rule trong `rules/generic/` (01-21):

1. Read rule file → hiểu intent, severity, search patterns gợi ý
2. Apply lên files trong scope (dùng grep + read tool của Codex)
3. Với mỗi match: trace data flow (L1-L4), phân loại có phải vulnerability thật không
4. Nếu có rule cùng `id` trong `rules/languages/<detected-lang>/`, **rule chuyên sâu thắng generic**.

**21 rules generic:**

| # | ID | Severity max |
|---|---|---|
| 1 | HARDCODED-SECRET | CRITICAL |
| 2 | SQL-INJECTION | CRITICAL |
| 3 | XSS | HIGH |
| 4 | IDOR | HIGH |
| 5 | SLOPSQUATTING | CRITICAL |
| 6 | BRUTE-FORCE | HIGH |
| 7 | MASS-ASSIGNMENT | CRITICAL |
| 8 | INSECURE-DESERIALIZATION | CRITICAL |
| 9 | SSRF | HIGH |
| 10 | PATH-TRAVERSAL | HIGH |
| 11 | CSRF | HIGH |
| 12 | BROKEN-ACCESS-CONTROL | CRITICAL |
| 13 | WEAK-PASSWORD-HASHING | CRITICAL |
| 14 | JWT-NONE-ALGORITHM | CRITICAL |
| 15 | CORS-MISCONFIG | HIGH |
| 16 | UNRESTRICTED-FILE-UPLOAD | CRITICAL |
| 17 | VERBOSE-ERROR-DEBUG-MODE | HIGH |
| 18 | MISSING-RATE-LIMIT | HIGH |
| 19 | RACE-CONDITION | HIGH |
| 20 | OUTDATED-DEPENDENCY | HIGH |
| 21 | COMMAND-INJECTION | CRITICAL |

---

## Step 5: Generate Report

Tham khảo template trong [`references/output-format.md`](references/output-format.md). Quy tắc cốt lõi:

**Verbose level theo severity:**
- **CRITICAL** → bảng overview + full verbose block per finding (Mô tả ngắn + Tại sao nguy hiểm + Hacker khai thác + Code before/after + Đọc thêm)
- **HIGH** → bảng overview + medium block per finding (Mô tả + Tác động + Code fix + Đọc thêm)
- **MEDIUM** → chỉ bảng compact
- **LOW** → chỉ bảng compact

**Layout:**
1. Header block (scope, file count, primary lang, mode, date, lang code)
2. VERDICT + 1-line description
3. CRITICAL section (overview table → verbose blocks)
4. HIGH section (overview table → medium blocks)
5. MEDIUM section (compact table)
6. LOW section (compact table)
7. PASSED CHECKS (list)
8. Next steps
9. **Save notification** (path file đã ghi)
10. **Gitignore warning** (nếu cần)
11. Footer + disclaimer
12. JSON summary (canonical EN — không phụ thuộc lang)

**Save-to-file:** ghi TOÀN BỘ report (identical với stdout) vào `vbsec-reports/scan-<timestamp>.md` dùng tool write/create file của Codex.

Sau đó in 1-2 dòng note ra stdout:
```
📄 {msg_report_saved}: vbsec-reports/scan-<timestamp>.md
⚠️ {msg_gitignore_warning_title}: {msg_gitignore_warning_text}
```

Mọi section header, severity label, verdict text lấy từ i18n file đã load ở Step 1.

---

## Verdict Logic

| Điều kiện | Verdict |
|---|---|
| Có ≥1 CRITICAL | **FAIL** |
| Không CRITICAL, có ≥1 HIGH | **WARN** |
| Không CRITICAL, không HIGH | **PASS** |

WARN ≠ approve. Báo cáo cần nêu rõ HIGH issues cần khắc phục trước production.

---

## Khác biệt với Claude Code variant

| Aspect | Claude Code | Codex (file này) |
|---|---|---|
| LARGE mode | Parallel sub-agents (3 cùng lúc) | Sequential chunking (1 chunk/lần) |
| Resume on interrupt | TodoWrite tasks | `.vbsec-tmp/findings-*.md` (re-run skip chunk đã có file) |
| Frontmatter | `userInvocable: true` | `license: MIT` |
| Invocation | `/vbs-scan-security` | `/skills` → pick, hoặc `$vbs-scan-security`, hoặc auto-match |

Toàn bộ rules, i18n, output format, language detection — identical với Claude Code variant. Khi update rule → sửa ở canonical (`skills/vbs-scan-security/`) → chạy `./scripts/sync-skills.sh` để propagate.

---

## Reasoning-First (cốt lõi)

**DO:**
- Đọc full function khi gặp pattern, KHÔNG flag luôn
- Trace nguồn dữ liệu: input → transformations → sink
- Phân loại L1-L4 trước khi flag CRITICAL
- Đọc rule file trước khi áp dụng

**DON'T:**
- Copy bash example chạy thẳng (đó là minh họa)
- Flag mọi `fmt.Sprintf` là SQLi (chỉ flag nếu data là L1 và không parameterize)
- Bỏ qua "but" clauses (nhiều pattern legitimate)
- Skip context (1 dòng grep không đủ để verdict)

**Mục tiêu là hiểu bảo mật, không phải đếm pattern.**
