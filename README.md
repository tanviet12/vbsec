<div align="center">

### 🇻🇳 [Đọc bằng Tiếng Việt → README.vi.md](README.vi.md)

</div>

---

# vbsec — Source Code Security Scanner

A multi-platform agent skill that performs in-depth security scans and detects 20+ of the most common security vulnerabilities in your source code. Runs natively on **Claude Code**, **OpenAI Codex CLI**, and **Google Antigravity**.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Claude Code](https://img.shields.io/badge/Claude%20Code-Skill-blue)](https://docs.claude.com/claude-code)
[![OpenAI Codex](https://img.shields.io/badge/OpenAI%20Codex-Skill-black)](https://developers.openai.com/codex/skills)
[![Google Antigravity](https://img.shields.io/badge/Google%20Antigravity-Skill-orange)](https://antigravity.google/docs/skills)

---

## Introduction

AI-generated code now represents a meaningful share of new commits across the industry. While modern coding assistants excel at producing code that *works*, they routinely ship code with classic security pitfalls: hardcoded secrets, SQL injection, missing access controls, weak password hashing, JWT misuse, and broken CORS configurations. These mistakes rarely surface in functional testing — they surface in incident reports.

vbsec brings production-grade security review into the AI coding loop. It runs as a native agent skill on three platforms — type `/vbs-scan-security` in Claude Code, `$vbs-scan-security` (or `/skills`) in OpenAI Codex CLI, or simply ask Google Antigravity to "scan security" — and receive a structured report covering 20+ categories of vulnerabilities. There are no external API calls, no separate tool installation, and no additional infrastructure to maintain.

vbsec has been exercised against intentionally vulnerable open-source training apps such as OWASP Juice Shop — and identifies findings that line up with the documented vulnerability challenges across SQL injection, NoSQL injection, JWT misuse, broken access control, mass assignment, deserialization RCE, and more.

Generic rules apply to every language. Specialized rule overlays exist for Go, PHP, TypeScript/JavaScript, Python, and .NET/C#, covering common frameworks: React, Vue, Angular, Express, NestJS, Next.js, Django, Flask, FastAPI, SQLAlchemy, Sequelize, Prisma, Mongoose, ASP.NET Core, and Entity Framework Core. Additional language overlays are on the roadmap.

## Authors

- **Bùi Tấn Việt** — CEO, [SePay](https://sepay.vn) & [123HOST](https://123host.vn)
- **Phan Quốc Hiên** — CTO, [SePay](https://sepay.vn) & [123HOST](https://123host.vn)

## How it works

vbsec is engineered around a small set of design choices that distinguish it from conventional pattern scanners.

- **Reasoning-first, not pattern counting.** vbsec does not blindly grep for `eval(` or `query(`. Each potential finding is verified by reading the surrounding code, tracing data flow (L1 untrusted user input through L4 trusted system data), and confirming the data reaches a dangerous sink without sanitization. This eliminates the false-positive flood typical of regex-based scanners.

- **Size-aware routing.** Small scans (≤20 main-language files AND ≤30 total) run inline in 30-60 seconds. Larger scans automatically delegate work to sub-agents that run in parallel — one chunk per top-level folder — and aggregate findings centrally. The user experience is identical; only the execution strategy changes.

- **Sub-agent delegation for large repositories.** For repositories with hundreds of files, vbsec spawns up to three parallel sub-agents through Claude Code's general-purpose agent. Each sub-agent scans a chunk of files independently, and findings are dedupe-aggregated by `(file, line, rule_id)`. This keeps wall-clock time bounded even on monorepos.

- **Language overlay system.** When vbsec detects the primary language, it loads language-specific rule files from `rules/languages/<lang>/` that override the generic rules for that language. This catches framework-specific patterns: Mongoose `$where` NoSQL injection, Angular `bypassSecurityTrustHtml`, Sequelize template-literal SQL, JWT algorithm confusion, Gin debug mode in production builds.

- **L1–L4 data flow classification.** Inputs are classified by trust level. A `db.query(\`SELECT ${x}\`)` call is only reported as a finding when `x` originates from L1 (user-controlled input) and reaches the SQL sink without parameterization. Constants, environment variables, and trusted-source data do not generate false positives.

- **One finding, one rule.** A line of code that triggers both IDOR and Race Condition produces two findings — never a comma-separated double tag. This keeps counts honest, reports auditable, and the trailing JSON summary machine-parseable.

- **Bilingual reports.** Vietnamese is the default; English is selected with `lang=en`. The JSON summary at the report tail is always canonical English for CI and tooling consumption.

- **Multi-platform.** One canonical rule set, three platform variants. Claude Code uses parallel sub-agents for large scans; Codex and Antigravity use sequential chunking with identical output. A single `sync-skills.sh` script keeps rule definitions in lock-step across all three.

## Multi-platform support

vbsec ships three variants from a single source of truth:

| Platform | Skill folder | Install target | LARGE mode strategy |
|---|---|---|---|
| Claude Code | `skills/vbs-scan-security/` | `~/.claude/skills/vbs-scan-security` | Parallel sub-agents (3 concurrent) |
| OpenAI Codex CLI | `skills/codex/vbs-scan-security/` | `~/.agents/skills/vbs-scan-security` | Sequential chunking |
| Google Antigravity | `skills/antigravity/vbs-scan-security/` | `~/.gemini/antigravity/skills/vbs-scan-security` | Sequential chunking |

All three share the same 21 rules, language overlays, i18n strings, and output format. Findings are identical; only execution strategy differs. Sequential variants are ~3× slower wall-clock than Claude Code's parallel mode on large repositories, but produce the same JSON summary and the same Markdown report.

Contributors: edit rules in `skills/vbs-scan-security/` (the canonical Claude folder), then run `./scripts/sync-skills.sh` to propagate to the Codex and Antigravity variants. Platform-specific files (`SKILL.md`, `workflows/large-review*.md`) are hand-maintained.

## Installation

vbsec auto-detects every supported platform you have installed and wires up the skill. Run:

```bash
git clone https://github.com/tanviet12/vbsec ~/vbsec
cd ~/vbsec
./scripts/install.sh         # auto-detect, install for what's present
./scripts/install.sh --all   # force install for all 3 platforms regardless
```

Detection logic:
- **Claude Code** — binary `claude` in PATH
- **OpenAI Codex CLI** — binary `codex` in PATH
- **Google Antigravity** — app at `/Applications/Antigravity.app` (macOS) OR CLI tool `agy` in PATH (installed via Antigravity IDE menu)

Antigravity is an IDE (like VS Code), not a CLI. For a brand-new Antigravity user, the folder `~/.gemini/antigravity/skills/` does not exist by default — the installer creates it for you.

The installer symlinks the appropriate skill folder into each platform's expected location. To update later:

```bash
cd ~/vbsec && git pull
```

(Symlinks pick up the new version automatically; restart the CLI / IDE if needed.)

**Manual install for a single platform:**

```bash
# Claude Code
ln -sfn ~/vbsec/skills/vbs-scan-security              ~/.claude/skills/vbs-scan-security

# OpenAI Codex CLI
ln -sfn ~/vbsec/skills/codex/vbs-scan-security        ~/.agents/skills/vbs-scan-security

# Google Antigravity
ln -sfn ~/vbsec/skills/antigravity/vbs-scan-security  ~/.gemini/antigravity/skills/vbs-scan-security
```

Verify the install on each platform:

```
Claude Code:   /vbs-scan-security
Codex:         $vbs-scan-security        (or /skills, then pick)
Antigravity:   "scan security cho repo này"  (auto-trigger by description)
```

See [docs/en/installation.md](docs/en/installation.md) for prerequisites, troubleshooting, and update procedures.

## Usage

The default scope is the entire repository. This is a deliberate change from earlier versions and matches how teams typically request a security audit.

```bash
/vbs-scan-security                       # scan entire folder (default)
/vbs-scan-security uncommitted           # only scan uncommitted changes
/vbs-scan-security pr id 42 lang=en      # scan a PR, report in English
/vbs-scan-security commit within 7days   # scan last 7 days of commits
```

**Works without git.** Vibe coders rarely init `git` before pasting AI-generated code into a folder. The default scope (`/vbs-scan-security`) walks the filesystem directly when no `.git/` is present — common build/vendored folders are excluded automatically. Git-specific scopes (`uncommitted`, `staged`, `commit within`, `commit id`, `pr id`) still require a git repository and will print a helpful message asking you to init git or fall back to the default scope.

Reports are saved to `vbsec-reports/scan-<timestamp>.md` inside the scanned folder for re-reading, sharing with reviewers, and attaching to remediation tickets.

See [docs/en/usage.md](docs/en/usage.md) for all options including `staged`, single-commit scans, and PR scanning via `gh`.

## Vulnerabilities vbsec detects

| # | Rule ID | Severity max | Specialized for |
|---|---|---|---|
| 1 | `HARDCODED-SECRET` | CRITICAL | — |
| 2 | `SQL-INJECTION` | CRITICAL | go, php, typescript, python, dotnet |
| 3 | `XSS` | HIGH | typescript |
| 4 | `IDOR` | HIGH | — |
| 5 | `SLOPSQUATTING` | CRITICAL | — |
| 6 | `BRUTE-FORCE` | HIGH | — |
| 7 | `MASS-ASSIGNMENT` | CRITICAL | typescript, python, dotnet |
| 8 | `INSECURE-DESERIALIZATION` | CRITICAL | go, php, typescript, python, dotnet |
| 9 | `SSRF` | HIGH | go, typescript, python |
| 10 | `PATH-TRAVERSAL` | HIGH | — |
| 11 | `CSRF` | HIGH | php, typescript, python |
| 12 | `BROKEN-ACCESS-CONTROL` | CRITICAL | — |
| 13 | `WEAK-PASSWORD-HASHING` | CRITICAL | — |
| 14 | `JWT-NONE-ALGORITHM` | CRITICAL | typescript, python |
| 15 | `CORS-MISCONFIG` | HIGH | typescript, python |
| 16 | `UNRESTRICTED-FILE-UPLOAD` | CRITICAL | — |
| 17 | `VERBOSE-ERROR-DEBUG-MODE` | HIGH | go, php, typescript, python |
| 18 | `MISSING-RATE-LIMIT` | HIGH | — |
| 19 | `RACE-CONDITION` | HIGH | — |
| 20 | `OUTDATED-DEPENDENCY` | HIGH | — |
| 21 | `COMMAND-INJECTION` | CRITICAL | go, php, typescript, python, dotnet |

The list currently contains 21 rules and will continue to expand.

## Documentation

- [Installation](docs/en/installation.md)
- [Usage](docs/en/usage.md)
- [Full rule catalog](docs/en/rules.md)
- [Contributing](docs/en/contributing.md)

## Roadmap

- v0.1 — Generic rule set + Go + PHP specialization + bilingual output ✅
- v0.2 — TypeScript/JavaScript specialization (Sequelize/Prisma/Mongoose, React/Vue/Angular, Express/NestJS/Next.js) ✅
- v0.3 — Default scope changed to full-repo, persistent reports, verbose per-finding explanations ✅
- v0.4 — Python specialization (SQLAlchemy/Django ORM SQLi, pickle/yaml deserialization RCE, Werkzeug debugger, FastAPI/Flask/Django CSRF + CORS, PyJWT algorithms, subprocess shell=True) ✅
- v0.5 (current) — Multi-platform support: OpenAI Codex CLI + Google Antigravity (sequential LARGE mode, shared rule set, `install.sh` + `sync-skills.sh`) ✅; .NET/C# specialization (EF Core raw SQL, ASP.NET Core mass-assignment, Newtonsoft `TypeNameHandling` deserialization, `Process` command injection) ✅
- v0.6+ — Ruby, Java, Rust — community-driven

## Disclaimer

vbsec is a reference scanner. It catches common AI-generated code mistakes, but:

- It does NOT replace a professional security audit
- It does NOT guarantee 100% vulnerability coverage
- It does NOT fetch live CVE databases (run `npm audit` / `pip-audit` / `govulncheck` separately for that)

Use vbsec as a **first line of defense**, not as proof of security.

## License & Acknowledgments

Released under the [MIT License](LICENSE).

Built on the security expertise of [SePay](https://sepay.vn) and [123HOST](https://123host.vn) — two Vietnamese fintech and hosting companies that operate production systems under real-world threat conditions.

© 2026 Bùi Tấn Việt & Phan Quốc Hiên.
