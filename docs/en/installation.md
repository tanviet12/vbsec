# Installing vbsec

> vbsec is a Claude Code skill. Install in two commands.

---

## Table of contents

- [Requirements](#requirements)
- [Install](#install)
- [Verify](#verify)
- [Update](#update)
- [Uninstall](#uninstall)
- [After installation: report location & .gitignore](#after-installation-report-location--gitignore)
- [Troubleshooting](#troubleshooting)
- [Next steps](#next-steps)

---

## Requirements

- **Claude Code** installed (CLI or desktop)
- **A git repository** to scan (vbsec uses `git` to gather the file list per scope)
- **`gh` CLI** (optional) — only needed if you want to scan GitHub PRs via the `pr id <n>` scope

Quick check:

```bash
claude --version
git --version
gh --version        # (Optional) for the pr id scope
```

---

## Install

Run these two commands in your terminal (outside Claude Code):

```bash
git clone https://github.com/tanviet12/vbsec ~/vbsec
ln -sfn ~/vbsec/skills/vbs-scan-security ~/.claude/skills/vbs-scan-security
```

What each command does:

1. **`git clone`** — downloads the vbsec repository to `~/vbsec` (you can change the destination if you prefer; just adjust the second command accordingly).

2. **`ln -sfn`** — creates a symlink from `~/.claude/skills/vbs-scan-security` to the skill folder inside the clone. Claude Code auto-discovers anything under `~/.claude/skills/`. Using a symlink (instead of copying) means `git pull` updates the skill in place without having to re-copy.

After running the commands, **restart Claude Code** so it picks up the new skill.

---

## Verify

After restarting Claude Code, run:

```
/vbs-scan-security
```

If it prints a scan report (header `# vbsec Security Scan Report`), the install succeeded.

You can also confirm the skill is loaded by typing `/` in Claude Code — the autocomplete should list `vbs-scan-security`.

---

## Update

To pull the latest version:

```bash
cd ~/vbsec
git pull
```

Then restart Claude Code so it reloads the skill from disk.

---

## Uninstall

```bash
rm ~/.claude/skills/vbs-scan-security    # remove the symlink
rm -rf ~/vbsec                            # remove the source clone (optional)
```

Restart Claude Code.

---

## After installation: report location & .gitignore

When you first run `/vbs-scan-security` in a project repo, vbsec creates `vbsec-reports/scan-<timestamp>.md` in that repo to persist the report. To prevent committing these scan files, add to your project's `.gitignore`:

```bash
# Add to the scanned project's .gitignore:
echo "vbsec-reports/" >> .gitignore
git add .gitignore && git commit -m "Add vbsec-reports/ to .gitignore"
```

> ⚠️ vbsec prints a recommendation at the end of each scan if it detects `vbsec-reports/` is missing from `.gitignore`. It does **not** automatically modify `.gitignore` — that's your decision.

If you actually want to commit scan reports (e.g., for a compliance trail or PR review evidence), simply don't add the entry. The file naming is timestamped so multiple scans never collide.

---

## Troubleshooting

### `/vbs-scan-security` does not appear in Claude Code

- Confirm the symlink exists: `ls -la ~/.claude/skills/vbs-scan-security` should show a link to `~/vbsec/skills/vbs-scan-security`.
- Confirm the target exists: `ls ~/vbsec/skills/vbs-scan-security/SKILL.md` should print a file path.
- Restart Claude Code (close and reopen). Skill discovery runs on startup.

### "Permission denied" on the symlink

Check directory permissions:

```bash
ls -ld ~/.claude/skills
```

If the directory does not exist, create it: `mkdir -p ~/.claude/skills`.

### Multiple Claude Code workspaces

Skills installed in `~/.claude/skills/` are **global** — they apply to every Claude Code workspace you open. No per-workspace install is needed.

### `pr id <n>` scope errors with `command not found: gh`

The PR scope needs the GitHub `gh` CLI:

```bash
# macOS
brew install gh

# Ubuntu/Debian
sudo apt install gh

# Then authenticate
gh auth login
```

Workaround: use a different scope (`uncommitted`, `staged`, `commit id`, `all`).

### Windows / WSL

Native Windows is not officially supported. Use WSL2 with the same instructions above (the symlink and Claude Code's skill discovery both work in WSL).

---

## Next steps

- Read [usage.md](usage.md) for every scope and how to interpret the report
- See [rules.md](rules.md) for the 30 detection rules
- Want to contribute? See [contributing.md](contributing.md)
