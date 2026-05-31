# Contributing to vbsec

Thanks for your interest in vbsec! This guide covers the workflow for adding new rules, language specializations, or improving existing ones.

> Audience: developers comfortable with git + basic security knowledge. You do NOT need to know the Claude Code SDK — the skill is just markdown.

---

## Table of contents

- [Setting up your dev environment](#setting-up-your-dev-environment)
- [Welcome contribution types](#welcome-contribution-types)
- [General workflow](#general-workflow)
- [Adding a new generic rule](#adding-a-new-generic-rule)
- [Adding a language specialization](#adding-a-language-specialization)
- [Adding an i18n string](#adding-an-i18n-string)
- [Testing before you PR](#testing-before-you-pr)
- [Coding style for rule files](#coding-style-for-rule-files)
- [PR checklist + commit conventions](#pr-checklist--commit-conventions)

---

## Setting up your dev environment

```bash
# Fork the repo on GitHub first, then:
git clone git@github.com:<your-username>/vbsec.git ~/vbsec-dev
cd ~/vbsec-dev
git remote add upstream https://github.com/tanviet12/vbsec.git

# Symlink the skill folder so edits are testable immediately
mkdir -p ~/.claude/skills
ln -sfn ~/vbsec-dev/skill ~/.claude/skills/vbs-scan-security

# Restart Claude Code
```

Edit any file under `skill/` → open Claude Code → invoke `/vbs-scan-security` → your change is live (Claude Code reads files fresh on every skill invocation).

**Tip:** Create a separate test repo to dogfood:

```bash
mkdir ~/vbsec-test && cd ~/vbsec-test
git init
# Place unsafe files here to exercise rules
```

---

## Welcome contribution types

| Type | Priority | Touches |
|---|---|---|
| New language specialization (Ruby, JS/TS, Python, Java, Rust) | High | `skill/rules/languages/<lang>/` |
| Improving an existing rule (more patterns, fewer false positives) | High | `skill/rules/generic/NN-*.md` |
| Fixing reasoning bugs in a rule | High | `skill/rules/generic/NN-*.md` |
| Adding test cases (positive + negative) | Medium | `tests/` (no infra yet — bundle in PR description) |
| Adding a brand-new rule (22, 23...) | Medium | Discuss via issue first |
| CVE list updates for OUTDATED-DEPENDENCY | Medium | `skill/rules/generic/20-outdated-dependency.md` |
| Docs (typos, examples, workflow improvements) | Low-Medium | `docs/`, `README.*.md` |

**Before large PRs (new rule, new language):** open an issue first to avoid duplication of effort.

---

## General workflow

```
1. Fork + clone + symlink
       ↓
2. Branch: feat/add-ruby-specialization
       ↓
3. Edit rule files under skill/
       ↓
4. Test on a test repo (positive + negative case)
       ↓
5. Update docs/vi/rules.md + docs/en/rules.md
       ↓
6. Update i18n if you introduced new strings
       ↓
7. Commit using conventions (feat/fix/docs/lang)
       ↓
8. Push, open PR, include test plan
```

---

## Adding a new generic rule

### 1. Pick a name + number

The next rule number is **22**. ID is `UPPERCASE-KEBAB-CASE`, e.g.: `OPEN-REDIRECT`, `XML-XXE`, `LDAP-INJECTION`.

File: `skill/rules/generic/22-open-redirect.md`

### 2. Frontmatter template

```markdown
---
id: OPEN-REDIRECT
severity_max: HIGH
applies_to: all
---

# OPEN-REDIRECT — Short human-readable name

## Intent

(One paragraph: why this rule matters, real-world exploit scenario)

## When CRITICAL / HIGH / MEDIUM

| Condition | Severity |
|---|---|
| <condition 1> | CRITICAL |
| <condition 2> | HIGH |
| <condition 3> | MEDIUM |

## Reasoning

(Guidance for the LLM agent: how to decide, trace L1-L4, read context, etc.)

## Search patterns (illustrative — NOT literal commands)

```bash
# Example pattern for the Grep tool
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

(Short note used in the "Fix" column of the report)

## Cross-references

- OWASP: <link>
- CWE: CWE-601 (Open Redirect)
- Related rules: SSRF, CSRF
```

### 3. Update dependent files

| File | What to do |
|---|---|
| [`skill/SKILL.md`](../../skills/vbs-scan-security/SKILL.md) | Add a row to the rules table in Step 4 (renumber from "21 rules" to "22 rules") |
| [`docs/vi/rules.md`](../vi/rules.md) | Add a `### Rule 22 — OPEN-REDIRECT` section |
| [`docs/en/rules.md`](rules.md) | Same |
| [`README.vi.md`](../../README.vi.md) | Update the 21→22 list (the table) |
| [`README.en.md`](../../README.en.md) | Same |

### 4. Test

See [Testing before you PR](#testing-before-you-pr).

---

## Adding a language specialization

A specialization overrides a generic rule for a specific language with patterns + reasoning tuned to the popular frameworks/libraries of that language.

### Example: adding Ruby

```bash
mkdir -p skill/rules/languages/ruby
```

Pick the rules worth specializing (not all 21 — focus on ones with idiomatic Ruby patterns). Examples:

- SQL-INJECTION → ActiveRecord `.where("...")` raw vs parameterized
- MASS-ASSIGNMENT → `User.update(params)` missing `permit`
- INSECURE-DESERIALIZATION → `Marshal.load`, `YAML.load`

### Specialization file template

File: `skill/rules/languages/ruby/02-sql-injection.md` (number + name match the generic counterpart):

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

(Override / extend the generic rule's reasoning for Ruby idioms)
```

### Update dependent files

| File | What to do |
|---|---|
| [`skill/references/language-detection.md`](../../skills/vbs-scan-security/references/language-detection.md) | Add Ruby to the Phase 1 table (extension `.rb`, file `Gemfile`) |
| [`skill/SKILL.md`](../../skills/vbs-scan-security/SKILL.md) | The line "Phase 1 hiện hỗ trợ chuyên sâu: `go`, `php`" — add `ruby` |
| [`docs/vi/rules.md`](../vi/rules.md) | Update the "Specialization" column for rules with Ruby override |
| [`docs/en/rules.md`](rules.md) | Same |
| [`README.{vi,en}.md`](../../README.en.md) | Update Roadmap (mark Ruby done) |

### Folder README

You can add `skill/rules/languages/ruby/README.md` describing: frameworks covered (Rails, Sinatra), ORM (ActiveRecord), idiomatic Ruby conventions.

---

## Adding an i18n string

All user-facing strings in the report must go through i18n. When you need a new key:

1. Open `skill/references/i18n/vi.md` — add the key + Vietnamese value
2. Open `skill/references/i18n/en.md` — add the SAME key + English value
3. In the rule / template file, reference the key (do not hardcode the string)

**Rules:**
- Keys use `snake_case` (`fix_recommendation`, `verdict_fail`)
- NEVER translate: rule IDs, file paths, code snippets, command names
- Both vi/en files must share the same set of keys — missing one is a bug

---

## Testing before you PR

vbsec has no automated test runner yet. Manual testing on two cases:

### Positive case — rule MUST flag

```bash
mkdir ~/vbsec-test-positive && cd ~/vbsec-test-positive
git init

# Create an unsafe file for the new rule
cat > app.py <<'EOF'
import os
filename = request.args.get('file')
os.system(f"convert {filename} output.png")  # command injection
EOF

git add app.py

# In a Claude Code session in this directory:
#   /vbs-scan-security staged
# Report MUST contain CRITICAL with rule COMMAND-INJECTION at app.py:3
```

### Negative case — rule MUST NOT false-positive

```bash
mkdir ~/vbsec-test-negative && cd ~/vbsec-test-negative
git init

# Safe version of the same code
cat > app.py <<'EOF'
import subprocess
subprocess.run(['convert', filename, 'output.png'], check=True)
EOF

git add app.py

# /vbs-scan-security staged
# Report MUST have COMMAND-INJECTION in "PASSED CHECKS" (no CRITICAL/HIGH)
```

### Edge cases — legitimate patterns

Many patterns look vulnerable but are safe in context. Example: `f"SELECT * FROM users WHERE id={user_id}"` is SAFE if `user_id` is an internal L3 variable (hardcoded const, framework-provided). Make sure your rule doesn't flag these.

### Document the test plan in the PR

Include in the PR description:

```markdown
## Test plan

- [x] Positive: app/sql_injection_unsafe.rb → flagged CRITICAL SQL-INJECTION
- [x] Negative: app/sql_injection_safe.rb → no finding
- [x] Edge: app/sql_injection_internal_const.rb → no finding (L3 input)
```

---

## Coding style for rule files

- **Prose language:** Vietnamese for explanations, English for rule IDs + code identifiers
- **Code snippets:** keep accurate language tags (` ```python `, ` ```javascript `, ` ```go `, ` ```php `)
- **No emoji** except checkmarks `✓` / `✗` in tables (keeps the format machine-parseable)
- **Length:** rule files between 100-300 lines. Shorter = missing context; longer = bloats LLM context window.
- **Bash patterns in rules are ILLUSTRATIVE:** write "illustrative" so the LLM agent doesn't run them literally. The skill uses Grep/Read tools, not raw Bash grep.
- **Cross-references:** each rule has OWASP + CWE + related rule links

### Required heading hierarchy

```markdown
# RULE-ID — Human-readable name

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

### Pre-PR checklist

- [ ] Branch from latest `main`
- [ ] Rule file has full frontmatter (id, severity_max, applies_to)
- [ ] Positive case tested → flags correctly
- [ ] Negative case tested → does NOT false-positive
- [ ] `docs/vi/rules.md` + `docs/en/rules.md` updated
- [ ] `SKILL.md` Step 4 table updated if adding a new rule
- [ ] README.vi.md + README.en.md updated if adding a new rule
- [ ] If adding new strings: both `i18n/vi.md` AND `i18n/en.md` updated
- [ ] Commit message follows convention below
- [ ] PR description includes test plan

### Commit conventions

| Prefix | When to use | Example |
|---|---|---|
| `feat:` | Add a feature / new rule | `feat: add OPEN-REDIRECT rule (22)` |
| `fix:` | Bug fix, false-positive reduction | `fix: reduce false positive in SQL-INJECTION for parameterized GORM` |
| `docs:` | Docs / README updates | `docs: clarify SMALL vs LARGE thresholds` |
| `lang:` | Add/modify a language specialization | `lang: add Ruby specialization for SQL-INJECTION` |
| `refactor:` | Restructure rule file without behavior change | `refactor: split JWT rule into 3 sub-checks` |
| `chore:` | Build, CI, formatting | `chore: add markdownlint config` |

Format: `<type>: <imperative summary>` (≤72 chars), blank line + body if needed.

### PR title

Use the same convention: `feat: add Ruby specialization`. The description should include:

```markdown
## What

(1-2 sentence summary)

## Why

(Why is this change needed?)

## Test plan

- [ ] Positive case ...
- [ ] Negative case ...

## Checklist

- [ ] docs/vi/rules.md updated
- [ ] docs/en/rules.md updated
- [ ] i18n updated (if applicable)
```

---

## FAQ

**Q: Do I need to know the Claude Code SDK?**
A: No. The skill is pure markdown. You only need to write .md and understand each rule's intent.

**Q: How do I test without burning API tokens?**
A: Make a tiny test repo (1-3 files), use SMALL mode (~30s per scan). Avoid the `all` scope while developing.

**Q: Are the bash patterns in rules executable?**
A: They're illustrative for the LLM, NOT commands to execute. The skill uses Claude Code's Grep tool.

**Q: Can I contribute a new rule without updating SKILL.md's table?**
A: No. That table is the source of truth — Claude Code reads SKILL.md to know which rules to apply.

**Q: Any other questions?**
A: Open an [issue](https://github.com/tanviet12/vbsec/issues) or a discussion on GitHub.
