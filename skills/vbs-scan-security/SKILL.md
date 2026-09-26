---
name: vbs-scan-security
description: Use when scanning code for security vulnerabilities. Use when user says "scan security", "kiểm tra bảo mật", "security audit", "review security", or invokes `/vbs-scan-security`. Auto-delegates to sub-agents for large scans (>20 main-language files OR >30 total OR >14 days). Outputs bilingual reports (vi/en). Optional `--auto-fix` (agentic patch + verify loop) and `--sca` (live CVE lookup via OSV.dev).
userInvocable: true
---

# vbsec — Security Scanner cho Vibe Coders

Quét lỗ hổng bảo mật cho code do AI sinh ra (vibe code). Bộ skill này check 21 lỗi bảo mật phổ biến nhất của vibe code, kế thừa kiến trúc SMALL/LARGE mode từ bộ rule production của SePay, tổng quát hóa cross-language (mặc định) + chuyên sâu cho Go/PHP (phase 1).

> Public repo: https://github.com/tanviet12/vbsec
> License: MIT (sẽ chốt khi public)

## Invocation

| Command | Scope | Mô tả |
|---|---|---|
| `/vbs-scan-security` | **Toàn repo** (default từ v0.3) | Mặc định — quét toàn bộ repo |
| `/vbs-scan-security all` | Toàn repo | Alias explicit của default |
| `/vbs-scan-security uncommitted` | Uncommitted changes | Quét staged + unstaged (cần explicit từ v0.3) |
| `/vbs-scan-security diff` | Uncommitted changes | Alias intuitive cho `uncommitted` |
| `/vbs-scan-security staged` | Staged files only | Pre-commit scan |
| `/vbs-scan-security commit within Xdays` | Recent commits | Quét commit X ngày gần đây |
| `/vbs-scan-security commit id <sha>` | Specific commit | Quét 1 commit |
| `/vbs-scan-security pr id <number>` | Pull request | Quét PR diff (cần `gh` CLI) |

**v0.3 change:** Default scope đổi từ `uncommitted` → `all`. Non-tech user lần đầu chạy không bị confused bởi report rỗng. Để giữ behavior cũ, dùng `uncommitted` hoặc `diff` explicit.

**Lựa chọn ngôn ngữ output (thêm vào bất kỳ scope nào):**
- `lang=vi` hoặc `--vi` → Tiếng Việt (mặc định)
- `lang=en` hoặc `--en` → English

**Cờ tùy chọn (v0.7+, mặc định TẮT — thêm vào bất kỳ scope nào):**

| Flag | Alias | Mô tả |
|---|---|---|
| `--sca` | `sca` | Tra cứu CVE **live** qua OSV.dev cho dependency (5 ecosystem: NuGet/Go/npm/Composer/PyPI). Xem [`references/dependency-scan.md`](references/dependency-scan.md). Cần network. |
| `--auto-fix` | `auto-fix` | Tự sinh patch (unified diff) cho finding CRITICAL/HIGH, verify bằng build command, revert nếu fail. Xem [`workflows/auto-fix.md`](workflows/auto-fix.md). **Ghi đè file nguồn** — cần git repo, khuyến nghị working tree sạch trước khi chạy. |

Ví dụ:
```
/vbs-scan-security pr id 42 lang=en
/vbs-scan-security staged --vi
/vbs-scan-security commit within 7days
/vbs-scan-security all --sca
/vbs-scan-security uncommitted --auto-fix
/vbs-scan-security all --sca --auto-fix
```

---

## CRITICAL: Cách dùng skill này (cho LLM agent)

**Các pattern bash/grep trong rule files là VÍ DỤ minh họa, KHÔNG phải lệnh chạy literal.**

### Nguyên tắc

1. **Lý luận, không pattern-match thuần** — Hiểu intent bảo mật đằng sau mỗi check, không chỉ tìm chuỗi
2. **Dùng tool phù hợp** — `Grep`, `Read`, `Glob` thay vì bash grep/find
3. **Đọc context đầy đủ** — Khi gặp pattern, READ hàm xung quanh để hiểu đây có thực sự là lỗ hổng không
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
│                     vbsec SCAN WORKFLOW                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  [Step 0] Parse args                                                 │
│     ├─ Scope (uncommitted/staged/commit/pr/all)                      │
│     └─ Output lang (vi default | en)                                 │
│                  ↓                                                   │
│  [Step 1] Gather files (git)                                         │
│                  ↓                                                   │
│  [Step 2] Detect primary code language                               │
│     └─ Đọc references/language-detection.md                          │
│                  ↓                                                   │
│  [Step 3] Route by size                                              │
│     ┌──────────────────┬──────────────────┐                          │
│     │  SMALL (inline)  │  LARGE (delegate)│                          │
│     │  ≤20 main+≤30tot │  >20 OR >30 OR   │                          │
│     │  AND ≤14d        │  >14 ngày        │                          │
│     └─────┬────────────┴─────────┬────────┘                          │
│           ↓                      ↓                                   │
│   workflows/small-     workflows/large-                              │
│   review.md            review.md                                     │
│                                                                      │
│   Both apply:                                                        │
│     - rules/generic/*.md (21 rules cross-language, luôn chạy)        │
│     - rules/languages/<detected>/*.md (override nếu trùng tên)       │
│                  ↓                                                   │
│  [Step 4b] SCA scan (optional — cần $SCA=true)                       │
│     └─ references/dependency-scan.md → rule 22 VULNERABLE-DEPENDENCY │
│                  ↓                                                   │
│  [Step 4c] Auto-fix (optional — cần $AUTO_FIX=true)                  │
│     └─ workflows/auto-fix.md → patch + verify + retry loop           │
│        (chạy TRƯỚC khi render report cuối, để patch_status kịp vào   │
│         JSON summary + section Auto-fix trong report)                │
│                  ↓                                                   │
│  [Step 5] Generate report                                            │
│     ├─ Markdown report (theo lang chọn, gồm cả section Auto-fix)     │
│     └─ JSON summary (canonical EN, ở cuối, gồm patch_status)         │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Step 0: Parse Arguments

Dùng Bash tool ĐÚNG MỘT LẦN cho step này (gather files là việc của git, không phải reasoning).

```bash
ARGS="${ARGUMENTS:-}"

# 0) Detect git availability (KHÔNG bắt buộc có git — v0.5.1+)
IS_GIT_REPO=true
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || IS_GIT_REPO=false

# 1) Extract lang flag (default vi)
LANG="vi"
if echo "$ARGS" | grep -qE 'lang=en|--en|\ben\b'; then LANG="en"; fi
if echo "$ARGS" | grep -qE 'lang=vi|--vi'; then LANG="vi"; fi

# 1b) Extract --auto-fix / --sca flags (v0.7+, default off) + scope.
#     Duyệt từng từ thay vì sed \b — BSD sed trên macOS không hỗ trợ \b.
AUTO_FIX=false
SCA=false
SCOPE_WORDS=""
set -f  # không expand glob khi tách từ
for w in $ARGS; do
  case "$w" in
    --auto-fix|auto-fix)       AUTO_FIX=true ;;
    --sca|sca)                 SCA=true ;;
    lang=vi|lang=en|--vi|--en) ;;
    *)                         SCOPE_WORDS="$SCOPE_WORDS $w" ;;
  esac
done
set +f

# 2) Scope = các từ còn lại (đã bỏ lang + auto-fix/sca)
SCOPE=$(echo "$SCOPE_WORDS" | xargs)

# 3) Gather files
NO_GIT_NOTE=""
SCAN_REF=""
SCAN_ROOT="."
case "$SCOPE" in
  "staged"|"uncommitted"|"diff"|"commit within "*|"commit id "*|"pr id "*)
    if [ "$IS_GIT_REPO" = false ]; then
      echo "{msg_scope_needs_git}"
      exit 1
    fi
    case "$SCOPE" in
      "staged")             FILES=$(git diff --cached --name-only --diff-filter=d) ;;
      "uncommitted"|"diff")
        # staged + unstaged (so với HEAD) + file mới chưa `git add`; bỏ file đã xoá
        FILES=$( { git diff --name-only --diff-filter=d HEAD 2>/dev/null || git diff --cached --name-only --diff-filter=d; git ls-files --others --exclude-standard; } | sort -u | grep -v '^$' || true) ;;
      "commit within "*)
        DAYS=$(echo "$SCOPE" | grep -oE '[0-9]+')
        # Đọc bản hiện tại trên đĩa → bỏ file đã bị xoá sau đó
        FILES=$(git log --since="${DAYS} days ago" --name-only --pretty=format: | sort -u | grep -v '^$' | while IFS= read -r f; do [ -f "$f" ] && echo "$f"; done || true) ;;
      "commit id "*)
        SHA=$(echo "$SCOPE" | sed 's/commit id //')
        git cat-file -e "${SHA}^{commit}" 2>/dev/null || { echo "Unknown commit: $SHA"; exit 1; }
        FILES=$(git diff-tree --root --no-commit-id --name-only -r --diff-filter=d "$SHA")
        SCAN_REF="$SHA" ;;
      "pr id "*)
        PR=$(echo "$SCOPE" | sed 's/pr id //')
        FILES=$(gh pr diff "$PR" --name-only) || exit 1
        git fetch -q origin "pull/${PR}/head" 2>/dev/null || git fetch -q "$(gh repo view --json url -q .url)" "pull/${PR}/head" || { echo "Cannot fetch PR #$PR"; exit 1; }
        SCAN_REF=$(git rev-parse FETCH_HEAD)
        # Bỏ file PR đã xoá (không còn ở head của PR)
        FILES=$(echo "$FILES" | while IFS= read -r f; do git cat-file -e "${SCAN_REF}:$f" 2>/dev/null && echo "$f"; done || true) ;;
    esac
    ;;
  "all"|"")
    if [ "$IS_GIT_REPO" = true ]; then
      FILES=$(git ls-files)
    else
      # Non-git folder — walk filesystem. Exclude folder system + vendored, GIỮ dot-files
      # như .env (để scan secrets), .htaccess, .gitignore (file thường, không phải folder).
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

# 3b) Scope theo commit/PR: extract snapshot đúng ref ra thư mục tạm.
#     Thư mục hiện tại có thể đang ở branch khác → đọc ở đó sẽ quét sai code.
if [ -n "$SCAN_REF" ]; then
  TMP_BASE="${TMPDIR:-/tmp}"; SCAN_ROOT=$(mktemp -d "${TMP_BASE%/}/vbsec-scan.XXXXXX")
  git archive "$SCAN_REF" | tar -x -C "$SCAN_ROOT"
fi

# 4) Strip noise (double-protect — vd git ls-files có thể trả file ở submodule vendored)
FILES=$(echo "$FILES" | grep -vE '(^|/)(node_modules|vendor|dist|build|\.next|\.nuxt|target|\.venv|__pycache__|\.git|vbsec-reports)/' || true)

# 5) Prepare save location (v0.3+)
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
echo "Scan root: $SCAN_ROOT"
echo "SCA (live OSV lookup): $SCA"
echo "Auto-fix: $AUTO_FIX"
[ "$NO_GIT_NOTE" = "true" ] && echo "Note: non-git folder — scanning all files via find"
[ "$GITIGNORE_WARNING" = "missing" ] && echo "Note: vbsec-reports/ not in .gitignore — will warn user at end"
```

**Quan trọng:**
- `vbsec-reports/` được excluded khỏi scan list — không scan chính báo cáo của mình
- **Scan root:** nếu `Scan root` khác `.` (scope `commit id`, `pr id`), mọi Read/Grep phải đọc file tại `$SCAN_ROOT/<path>`. Đó là snapshot đúng commit/PR; KHÔNG đọc bản trong thư mục hiện tại (có thể đang ở branch khác). Report vẫn ghi path gốc `<path>`, không kèm prefix `$SCAN_ROOT`. LARGE mode: truyền `$SCAN_ROOT` làm `{repo_path}` cho từng chunk. Render report xong → `rm -rf "$SCAN_ROOT"`.
- Path output `vbsec-reports/scan-<timestamp>.md` cần được mkdir trước khi scan, để workflows save vào
- **v0.5.1+**: skill chạy được trên cả non-git folder. Default scope (`all`) dùng `find` thay `git ls-files`. Các scope dựa vào git (`staged`, `uncommitted`, `commit within`, `commit id`, `pr id`) BẮT BUỘC git — báo `msg_scope_needs_git` rồi exit.
- Nếu `NO_GIT_NOTE=true`, report header phải in `{msg_no_git_note}` để user biết folder không có git → không lọc theo `.gitignore`.
- **v0.7+**: `$AUTO_FIX=true` nhưng `$IS_GIT_REPO=false` → Step 4c (auto-fix) sẽ tự skip và in `{msg_autofix_needs_git}` (không exit toàn bộ scan, phần scan/report vẫn chạy bình thường).
- **v0.7+**: `$AUTO_FIX=true` và `$SCAN_ROOT` khác `.` (scope `commit id`, `pr id`) → file đang đọc là snapshot tạm, sửa ở đó không có tác dụng. Step 4c KHÔNG apply patch nào: mọi finding CRITICAL/HIGH chỉ ghi diff ra `vbsec-reports/patches/`, `patch_status: "suggested_only"`, và in `{msg_autofix_snapshot_scope}` một lần.

---

## Step 1: Load i18n Strings

Đọc file i18n tương ứng với `$LANG`:
- `lang=vi` → Read [`references/i18n/vi.md`](references/i18n/vi.md)
- `lang=en` → Read [`references/i18n/en.md`](references/i18n/en.md)

File i18n chứa bảng key→text cho toàn bộ user-facing strings (section headers, severity labels, verdict, fix recommendations templates). Mọi text trong report final phải lấy từ i18n, KHÔNG hardcode.

**Strings KHÔNG bao giờ dịch:** rule ID (SQL-INJECTION, XSS, IDOR...), file path, code snippet, command name (`/vbs-scan-security`).

---

## Step 2: Detect Primary Code Language

Đọc [`references/language-detection.md`](references/language-detection.md) để biết cách detect. Tóm tắt:

1. Count extension trong file list (đã strip vendored): `.go`, `.py`, `.php`, `.js`, `.ts`, `.jsx`, `.tsx`, `.rb`, `.java`, `.rs`, `.cs`, `.csproj`, `.sln`
2. Primary lang = lang chiếm ≥30% tổng files
3. Có `rules/languages/<lang>/` → load overlay; không có → chỉ dùng generic
4. Multi-lang repo (cả Go backend + Vue frontend) → load cả 2 overlay

**Hiện hỗ trợ chuyên sâu:** `go`, `php`, `typescript` (gộp JS+TS), `python`, `dotnet`. Các lang khác chỉ dùng generic rules.

---

## Step 3: Route by Size

| Điều kiện | Ngưỡng | Mode |
|---|---|---|
| Files ngôn ngữ chính | ≤20 | SMALL |
| Files ngôn ngữ chính | >20 | **LARGE** |
| Tổng files | ≤30 | SMALL |
| Tổng files | >30 | **LARGE** |
| Timespan (chỉ với scope `commit within`) | ≤14 ngày | SMALL |
| Timespan | >14 ngày | **LARGE** |

BẤT KỲ điều kiện nào sang LARGE → dùng LARGE mode.

- **SMALL mode:** Read [`workflows/small-review.md`](workflows/small-review.md) và follow workflow đó (inline, không sub-agent)
- **LARGE mode:** Read [`workflows/large-review.md`](workflows/large-review.md), trở thành **orchestrator only**:
  1. TodoWrite cho từng chunk (resume được nếu interrupt)
  2. Chunk files theo top-level folder (xem [`references/chunking-strategy.md`](references/chunking-strategy.md))
  3. Spawn sub-agents (general-purpose) cho mỗi chunk với prompt từ [`references/sub-agent-prompts.md`](references/sub-agent-prompts.md)
  4. Sub-agents ghi findings ra `.vbsec-tmp/findings-<chunk>.md` (luôn dùng EN canonical + rule ID)
  5. Main agent aggregate → translate sang `$LANG` → final report
  6. Cleanup `.vbsec-tmp/` sau khi done

---

## Step 4: Apply Rules

Cho mỗi rule trong `rules/generic/` (01-21):

1. Nạp phần phát hiện của cả bộ rule qua script `bash <skill-dir>/references/load-rules.sh --part N <lang...>` (chạy đủ mọi phần, dòng cuối output cho biết tổng số phần). **Chạy nguyên lệnh, KHÔNG thêm `| head`, `| tail`, `| grep`**: mỗi phần đã < 20.000 ký tự, cắt output = bỏ sót rule (overlay đã thay generic) → hiểu intent, severity, search patterns gợi ý. Phần Examples/Fix recommendation chỉ Read khi rule có finding CRITICAL/HIGH (chi tiết trong workflow)
2. Apply lên files trong scope (dùng Grep/Read tool)
3. Với mỗi match: trace data flow (L1-L4), phân loại có phải vulnerability thật không
4. Nếu có rule cùng tên (cùng `id`) trong `rules/languages/<detected-lang>/`, **rule chuyên sâu thắng generic** (đè hoàn toàn pattern + reasoning steps cho lang đó).

**21 rules generic (luôn chạy) + 1 rule optional (`--sca`):**

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
| 22 | VULNERABLE-DEPENDENCY | CRITICAL |

Rule 22 chỉ chạy khi `$SCA=true` — xem Step 4b dưới đây. 21 rules còn lại luôn chạy.

---

## Step 4b: SCA Scan (optional — `--sca`)

Chỉ chạy khi `$SCA=true`. Đọc [`references/dependency-scan.md`](references/dependency-scan.md) và follow:

1. Parse manifest dependency theo ecosystem tương ứng `$PRIMARY_LANG` (NuGet/.NET, Go, npm/TypeScript, Composer/PHP, PyPI/Python) — có thể nhiều ecosystem nếu multi-lang repo.
2. Query live `https://api.osv.dev/v1/querybatch` rồi `v1/vulns/{id}` cho từng package (dùng Bash tool, đây là 1 trong 2 chỗ duy nhất trong skill được phép gọi network thật — chỗ còn lại là `gh pr diff`).
3. Map CVSS → severity, tạo finding `rule_id: VULNERABLE-DEPENDENCY` kèm `cve_id`/`fixed_version`.
4. Network fail/không có manifest → in `{msg_sca_unavailable}`/`{msg_sca_no_manifest}`, KHÔNG fail scan, fallback sang rule 20 (đã chạy sẵn ở Step 4).
5. Merge findings rule 22 vào cùng danh sách trước khi qua Step 5. Nếu trùng package/version với finding rule 20, chỉ giữ rule 22.
6. **LARGE mode:** đây là bước main agent tự chạy 1 lần cho toàn repo, KHÔNG delegate cho sub-agent theo chunk (dependency manifest không chia theo folder được) — xem [`references/sub-agent-prompts.md`](references/sub-agent-prompts.md) mục "Aggregate workflow".

---

## Step 4c: Auto-fix (optional — `--auto-fix`)

Chỉ chạy khi `$AUTO_FIX=true`. Chạy **TRƯỚC** khi render report ở Step 5 (để `patch_status` kịp có mặt trong JSON summary và section Auto-fix render đúng vị trí). Đọc [`workflows/auto-fix.md`](workflows/auto-fix.md) và follow toàn bộ workflow đó:

1. Gate: cần `$IS_GIT_REPO=true`, nếu không → in `{msg_autofix_needs_git}`, skip toàn bộ bước này (report vẫn render bình thường ở Step 5, chỉ thiếu section Auto-fix).
   Nếu `$SCAN_ROOT` khác `.` (scope `commit id`, `pr id`) → chỉ sinh patch, KHÔNG `git apply`/build verify; mọi finding CRITICAL/HIGH là `suggested_only` (xem gate trong workflow).
2. Chỉ xử lý finding CRITICAL/HIGH (từ cả rule 1-21 và rule 22 nếu có `--sca`).
3. Với mỗi finding: harvest context → generate unified diff → `git apply --check` → `git apply` → chạy build command theo `$PRIMARY_LANG` → revert + retry (tối đa 2 lần) nếu fail.
4. Gắn `patch_status` vào từng finding đã xử lý — dùng ở Step 5 khi render JSON + section Auto-fix.

---

## Step 5: Generate Report (v0.3+ — verbose + persistent)

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
7b. Hardening notes (tuỳ chọn, `{header_hardening_title}`) — gợi ý phòng thủ, KHÔNG phải finding
8. Next steps (1-2 dòng)
8b. **Auto-fix summary** (chỉ khi `--auto-fix` đã chạy ở Step 4c — xem [`workflows/auto-fix.md`](workflows/auto-fix.md))
9. **Save notification** (path file đã ghi)
10. **Gitignore warning** (nếu cần)
11. Footer + disclaimer
12. JSON summary (canonical EN — không phụ thuộc lang) — đúng schema ở `references/output-format.md` mục 7

**Save-to-file (v0.3+):**

Sau khi render report:

```bash
# Workflow đã chuẩn bị $REPORT_FILE và $GITIGNORE_WARNING ở Step 0
# Ghi TOÀN BỘ report (identical với stdout) vào file:
cat > "$REPORT_FILE" <<'REPORT_EOF'
<full report content here>
REPORT_EOF

# In dòng cuối ra stdout:
echo ""
# LLM thay {key} bằng giá trị từ i18n file đã load ở Step 1 (KHÔNG có shell function tên `i18n`):
echo "📄 {msg_report_saved}: $REPORT_FILE"
[ "$GITIGNORE_WARNING" = "missing" ] && echo "⚠️ {msg_gitignore_warning_title}: {msg_gitignore_warning_text}"
```

LLM agent thực thi bằng Write tool (NOT bash heredoc) để ghi file, sau đó in 1-2 dòng note ra stdout. Nội dung file PHẢI IDENTICAL với output trên stdout.

Mọi section header, severity label, verdict text lấy từ i18n file đã load ở Step 1.

**Finding vs hardening note:** chỉ tạo finding khi có đường khai thác cụ thể (input attacker điều khiển được tới sink, hoặc cấu hình sai khai thác được ngay). Reasoning kết luận "an toàn" → KHÔNG tạo finding. Gợi ý phòng thủ thêm cho code đã an toàn (header, cờ cookie khi không có XSS, lockfile...) → `hardening_notes[]` + section `{header_hardening_title}`, không gán `rule_id`, không tính vào summary/verdict. Chi tiết: [`references/output-format.md`](references/output-format.md) mục "Finding vs hardening note".

**Validate JSON trước khi kết thúc (bắt buộc):** sau khi ghi report, chạy `python3 <skill-dir>/references/validate-report.py <report-file>` (`<skill-dir>` = thư mục chứa file SKILL.md này). Script báo lỗi → sửa JSON trong report, ghi lại, chạy lại (tối đa 2 lần). Lỗi hay gặp: dùng key `id`/`rule` thay vì `rule_id`, tự đặt rule ID ngoài 21 rule, severity viết thường, `summary` đếm lệch với `findings`. Không có `python3` → tự đối chiếu với bảng schema ở `output-format.md` mục 7.

---

## Verdict Logic

| Điều kiện | Verdict |
|---|---|
| Có ≥1 CRITICAL | **FAIL** |
| Không CRITICAL, có ≥1 HIGH | **WARN** |
| Không CRITICAL, không HIGH | **PASS** |

WARN ≠ approve. Báo cáo cần nêu rõ HIGH issues cần khắc phục trước production.

---

## Cấu trúc skill (cho người contribute)

```
~/.claude/skills/vbs-scan-security/
├── SKILL.md                          # File này
├── workflows/
│   ├── small-review.md               # Inline scan (default cho repo nhỏ-vừa)
│   ├── large-review.md               # Sub-agent delegation
│   └── auto-fix.md                   # v0.7+: patch/verify/retry loop (--auto-fix)
├── rules/
│   ├── generic/                      # 22 rules cross-language
│   │   ├── 01-hardcoded-secret.md
│   │   ├── 02-sql-injection.md
│   │   ├── ... (đến 21, luôn chạy)
│   │   ├── 21-command-injection.md
│   │   └── 22-vulnerable-dependency.md   # v0.7+: chỉ chạy với --sca
│   └── languages/                    # Override chuyên sâu per language
│       ├── go/                       # GORM, slog, Colly...
│       ├── php/                      # mysqli/PDO, $_GET, eval/include, Laravel CSRF
│       └── README.md                 # Hướng dẫn add language mới
└── references/
    ├── chunking-strategy.md
    ├── sub-agent-prompts.md
    ├── language-detection.md
    ├── data-flow-classification.md
    ├── dependency-scan.md             # v0.7+: parser manifest + OSV.dev query (--sca)
    ├── output-format.md
    └── i18n/
        ├── vi.md
        └── en.md
```

**Thêm rule mới (cross-language):** tạo file số tiếp theo (23+) trong `rules/generic/`, frontmatter có `id`, `severity_max`, `applies_to: all`. Update bảng ở Step 4 trong file này.

**Thêm language specialization mới (e.g., Ruby):** tạo `rules/languages/ruby/<rule-id>.md` với cùng `id` như generic — sẽ tự override. Đọc `rules/languages/README.md` để biết template.

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
