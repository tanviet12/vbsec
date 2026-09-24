# Using vbsec

Detailed guide to invoking `/vbs-scan-security`, choosing scopes, understanding reports, and integrating into CI/CD.

> Haven't installed the skill yet? Read [installation.md](installation.md) first.

---

## Table of contents

- [Basic command](#basic-command)
- [All supported scopes](#all-supported-scopes)
- [Where reports are saved](#where-reports-are-saved)
- [Choosing output language](#choosing-output-language)
- [Anatomy of a report](#anatomy-of-a-report)
- [JSON summary for tooling](#json-summary-for-tooling)
- [Performance expectations](#performance-expectations)
- [Resuming after LARGE mode is interrupted](#resuming-after-large-mode-is-interrupted)
- [CI/CD integration](#cicd-integration)

---

## Basic command

Open Claude Code inside your git repo. In the chat input:

```
/vbs-scan-security
```

> **⚠️ v0.3 behavior change:** `/vbs-scan-security` (no arguments) now scans the **entire repository**. Previously this was equivalent to `uncommitted`. If you want the old behavior, use `/vbs-scan-security uncommitted` or `/vbs-scan-security diff`.

By default vbsec will:
1. Collect every file in the repo (full tree, not just uncommitted)
2. Detect the primary language (Go/PHP/JS/Python/...)
3. Route to SMALL or LARGE mode based on file count
4. Apply 21 generic rules + language overlay (if available)
5. Print a Markdown report + JSON summary to stdout
6. Save a copy of the report to `vbsec-reports/scan-<timestamp>.md` in the repo

---

## All supported scopes

| Command | What it scans |
|---|---|
| `/vbs-scan-security` | **Entire repo (default — changed in v0.3)** |
| `/vbs-scan-security uncommitted` | Only uncommitted changes (staged + unstaged) |
| `/vbs-scan-security diff` | Alias for `uncommitted` |
| `/vbs-scan-security staged` | Only staged files |
| `/vbs-scan-security commit within 7days` | Files touched in the last N days |
| `/vbs-scan-security commit id <sha>` | Files in a specific commit |
| `/vbs-scan-security pr id 42` | Files in PR #42 (GitHub) — needs `gh` CLI |
| `/vbs-scan-security all` | Explicit alias for entire repo |

### Realistic examples

```bash
# Default — scan the whole repo, report in Vietnamese, saved to file
/vbs-scan-security

# Before committing: scan only what changed
/vbs-scan-security uncommitted

# Pre-merge review: scan a PR with English report
/vbs-scan-security pr id 42 lang=en

# Periodic audit: scan recent commits
/vbs-scan-security commit within 30days

# Find the latest report
ls -lt vbsec-reports/ | head -5

# Diff two scans
diff vbsec-reports/scan-2026-05-13-100000.md vbsec-reports/scan-2026-05-14-100000.md
```

---

## Where reports are saved

From v0.3 onwards, **every scan saves a copy of the report to a file** in the scanned repo:

- Path: `vbsec-reports/scan-<YYYY-MM-DD-HHMMSS>.md`
- File naming format: `scan-YYYY-MM-DD-HHMMSS.md` — sorts chronologically by default
- The content is identical to what's printed to stdout — the file is for re-reading and sharing, stdout is for the immediate view
- If `vbsec-reports/` is not in `.gitignore`, vbsec prints a recommendation at the end of stdout — **vbsec does not auto-modify `.gitignore`**, that's your decision
- Reports can be deleted any time; they're not state files

Find the latest scan:

```bash
ls -lt vbsec-reports/ | head -5
```

Show the most recent report:

```bash
cat "$(ls -t vbsec-reports/scan-*.md | head -1)"
```

See [installation.md](installation.md#after-installation-report-location--gitignore) for how to add `vbsec-reports/` to `.gitignore`.

---

## Choosing output language

vbsec supports Vietnamese (default) and English. How to specify:

| Syntax | Result |
|---|---|
| (no flag) | Vietnamese |
| `lang=vi` | Vietnamese |
| `--vi` | Vietnamese |
| `lang=en` | English |
| `--en` | English |

Examples:

```
/vbs-scan-security pr id 42 lang=en
/vbs-scan-security commit within 14days --en
/vbs-scan-security staged
```

**Note:** The Markdown report changes by lang, BUT:
- Rule IDs (HARDCODED-SECRET, SQL-INJECTION...) are always EN
- File paths and code snippets stay verbatim
- The JSON summary at the end is ALWAYS EN canonical (so tooling can parse reliably)

---

## Anatomy of a report

A typical report:

```
# vbsec Security Scan Report

Scope: Uncommitted changes
Files: 12 (8 .ts, 4 .py)
Primary language: typescript
Mode: SMALL (inline scan)
Date: 2026-05-13

## VERDICT: FAIL
CRITICAL issues found. DO NOT deploy until fixed.

## CRITICAL (blocks deploy)
| File:Line | Rule | Issue | Fix |
|---|---|---|---|
| api/users.ts:42 | SQL-INJECTION | req.body.id concatenated into SQL | Use parameterized query |
| .env:1 | HARDCODED-SECRET | STRIPE_KEY=sk_live_... | Rotate key + .gitignore |

## HIGH (must fix)
| auth.ts:18 | WEAK-PASSWORD-HASHING | createHash('md5') | Use bcrypt/argon2 |

## MEDIUM
(none)

## PASSED CHECKS
- ✓ XSS — Vue templates auto-escape
- ✓ CORS — specific origin
- ✓ CSRF — token present on forms

## JSON Summary
```json
{
  "verdict": "FAIL",
  "scope": "uncommitted",
  "files_scanned": 12,
  "primary_language": "typescript",
  "mode": "SMALL",
  "counts": {"critical": 2, "high": 1, "medium": 0, "low": 0},
  "findings": [
    {"file": "api/users.ts", "line": 42, "rule": "SQL-INJECTION", "severity": "CRITICAL"},
    {"file": ".env", "line": 1, "rule": "HARDCODED-SECRET", "severity": "CRITICAL"},
    {"file": "auth.ts", "line": 18, "rule": "WEAK-PASSWORD-HASHING", "severity": "HIGH"}
  ]
}
```

### Report sections

| Section | Purpose |
|---|---|
| Header | Scope scanned, file count, primary language, mode (SMALL/LARGE), date |
| VERDICT | PASS / WARN / FAIL — decides whether you can deploy |
| CRITICAL | Blocks deploy, fix first |
| HIGH | Must fix before production |
| MEDIUM | Fix soon, doesn't block |
| PASSED CHECKS | Rules verified clean — proof of full coverage |
| JSON Summary | Machine-readable, for CI parsing |

### Verdict logic

| Condition | Verdict | Meaning |
|---|---|---|
| ≥1 CRITICAL | **FAIL** | Do not deploy |
| 0 CRITICAL, ≥1 HIGH | **WARN** | Can deploy, but plan a fix |
| 0 CRITICAL, 0 HIGH | **PASS** | OK to deploy |

WARN is **not** approval. Security/tech lead should still review HIGH issues.

---

## JSON summary for tooling

The JSON summary always sits at the end of the report, in a `json` fenced code block. Schema is stable:

```json
{
  "verdict": "PASS" | "WARN" | "FAIL",
  "scope": "uncommitted" | "staged" | "commit_within_Ndays" | "commit_id_<sha>" | "pr_<num>" | "all",
  "files_scanned": <int>,
  "primary_language": <string>,
  "mode": "SMALL" | "LARGE",
  "counts": {
    "critical": <int>,
    "high": <int>,
    "medium": <int>,
    "low": <int>
  },
  "findings": [
    {
      "file": <string>,
      "line": <int>,
      "rule": <string>,
      "severity": "CRITICAL" | "HIGH" | "MEDIUM" | "LOW"
    }
  ]
}
```

### Parsing JSON from the output

The report is Markdown — extract JSON with:

```bash
# Report saved to report.md
awk '/^```json$/,/^```$/' report.md | sed '1d;$d' > summary.json
jq '.verdict' summary.json
# "FAIL"
```

Or in Node.js:

```javascript
const fs = require('fs');
const md = fs.readFileSync('report.md', 'utf8');
const match = md.match(/```json\n([\s\S]*?)\n```/);
const summary = JSON.parse(match[1]);
if (summary.verdict === 'FAIL') process.exit(1);
```

---

## Performance expectations

| Mode | Trigger | Time | Resources |
|---|---|---|---|
| SMALL | ≤20 primary-lang files AND ≤30 total AND ≤14 days | 30-60 seconds | 1 agent, ~5-15 tool calls |
| LARGE | Any of the three thresholds exceeded | 5-15 minutes | Parallel sub-agents, one per chunk |

LARGE mode uses [chunking-strategy.md](../../skills/vbs-scan-security/references/chunking-strategy.md) to split files by top-level folder, with one sub-agent per chunk. The main agent aggregates findings + translates at the end.

### When does it use SMALL vs LARGE?

You don't have to think about it — SKILL.md routes automatically. But if you want to force:

- Narrow scope (`staged`, `commit id`) → usually SMALL
- Wide scope (`all`, `commit within 30days`) → LARGE
- Small repo (<30 files) → always SMALL even with `all`

---

## Resuming after LARGE mode is interrupted

LARGE mode uses `TodoWrite` to track each chunk. If you interrupt mid-run (Ctrl+C, crash, etc.):

1. Reopen Claude Code in the same directory
2. Re-run the original command (same scope, same lang)
3. The skill sees the existing `.vbsec-tmp/` directory with prior findings and resumes from the unfinished chunk

**Important:**
- Don't manually delete `.vbsec-tmp/` if you want to resume
- After a successful scan, vbsec cleans up `.vbsec-tmp/` itself
- Add `.vbsec-tmp/` to your repo's `.gitignore`:
  ```
  .vbsec-tmp/
  ```

---

## CI/CD integration

vbsec is a Claude Code skill — runs inside the agent CLI. Two integration shapes:

### A. Pre-commit hook (local, each dev runs it)

`.git/hooks/pre-commit`:

```bash
#!/usr/bin/env bash
# Requires Claude Code installed + vbsec plugin installed (/plugin install vbsec@vbsec)
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

**Note:** The hook only blocks on `verdict=FAIL` (CRITICAL present). WARN is informational.

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

      - name: Install vbsec plugin
        run: |
          # Inside the Claude Code session: install vbsec marketplace + plugin
          # (Run these via `claude --no-stream -p` if the CI image supports it,
          # or use claude-code's plugin pre-install env vars.)
          mkdir -p ~/.claude/plugins
          # Option 1: pre-warm by cloning the plugin into the cache
          git clone https://github.com/tanviet12/vbsec.git ~/.claude/plugins/cache/vbsec
          # Option 2 (recommended once Claude Code CI tooling matures):
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

### C. Custom policies

Parse the JSON summary and apply your own policy:

```bash
# Block when ≥3 HIGH issues exist
HIGH_COUNT=$(jq -r '.counts.high' summary.json)
if [ "$HIGH_COUNT" -ge 3 ]; then
  echo "Too many HIGH issues ($HIGH_COUNT). Blocking."
  exit 1
fi

# Or: block on specific rules only
CRITICAL_RULES=$(jq -r '.findings[] | select(.severity=="CRITICAL") | .rule' summary.json)
if echo "$CRITICAL_RULES" | grep -q "HARDCODED-SECRET"; then
  echo "Hardcoded secret detected — blocking deploy."
  exit 1
fi
```

---

## Next steps

- Read [rules.md](rules.md) for full details on the 21 rules
- Want to add a new rule or language? See [contributing.md](contributing.md)
