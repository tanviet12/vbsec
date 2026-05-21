# Language Specialization Rules

This folder contains **language-specific overrides** for the 21 generic rules in `rules/generic/`. When vbsec detects the primary language of a target repo, the corresponding language folder is consulted; any rule with a matching `id` in frontmatter REPLACES the generic version for that scan.

## Override mechanism

```
rules/
├── generic/
│   ├── 01-hardcoded-secret.md         (applies_to: all)
│   ├── 02-sql-injection.md            (applies_to: all)
│   ├── ...
│   └── 21-command-injection.md        (applies_to: all)
└── languages/
    ├── go/
    │   ├── 02-sql-injection.md        (applies_to: go)   ← overrides generic 02
    │   ├── 08-insecure-deserialization.md
    │   ├── 09-ssrf.md
    │   ├── 17-verbose-error-debug-mode.md
    │   └── 21-command-injection.md
    ├── php/
    │   ├── 02-sql-injection.md        (applies_to: php)
    │   ├── 08-insecure-deserialization.md
    │   ├── 11-csrf.md
    │   ├── 17-verbose-error-debug-mode.md
    │   └── 21-command-injection.md
    ├── typescript/
    │   ├── 02-sql-injection.md        (applies_to: typescript — gộp cả .js .ts)
    │   ├── 03-xss.md
    │   ├── 07-mass-assignment.md
    │   ├── 08-insecure-deserialization.md
    │   ├── 09-ssrf.md
    │   ├── 11-csrf.md
    │   ├── 14-jwt-none-algorithm.md
    │   ├── 15-cors-misconfig.md
    │   ├── 17-verbose-error-debug-mode.md
    │   └── 21-command-injection.md
    └── python/
        ├── 02-sql-injection.md        (applies_to: python — .py .pyw)
        ├── 07-mass-assignment.md
        ├── 08-insecure-deserialization.md
        ├── 09-ssrf.md
        ├── 11-csrf.md
        ├── 14-jwt-none-algorithm.md
        ├── 15-cors-misconfig.md
        ├── 17-verbose-error-debug-mode.md
        └── 21-command-injection.md
    └── dotnet/
        ├── 02-sql-injection.md        (applies_to: dotnet — .cs .csproj .sln)
        ├── 07-mass-assignment.md
        ├── 08-insecure-deserialization.md
        └── 21-command-injection.md
```

**Matching is by frontmatter `id`, not filename.** Filename numeric prefix is convention only, to aid navigation. If `rules/languages/go/02-sql-injection.md` has `id: SQL-INJECTION`, it overrides `rules/generic/02-sql-injection.md` (also `id: SQL-INJECTION`) when scanning a Go repo.

Rules without a language-specific override fall back to the generic version. So a Go scan still uses generic `01-hardcoded-secret.md`, `03-xss.md`, `12-broken-access-control.md`, etc. — only the 5 overrides above are replaced.

## Language detection

See `references/language-detection.md` for the detection heuristic. In short:

- Count files by extension (`.go`, `.php`, `.py`, `.js`, `.ts`, ...) under the target path
- Primary language = highest count (with minimum threshold, e.g. ≥ 5 files)
- For mixed repos (vd: Laravel + Vue), detection picks the **backend** language (server-side risk dominates)
- Multi-language detection is possible: agent can run both `php` and `js` rule sets and merge

## Supported languages

| Language | Folder | Override count | Status |
|---|---|---|---|
| Go | `go/` | 5 | Stable (v0.1) |
| PHP | `php/` | 5 | Stable (v0.1) |
| TypeScript/JavaScript | `typescript/` | 10 | Stable (v0.2) — includes both `.ts` and `.js` files |
| Python | `python/` | 9 | Stable (v0.4) — `.py` and `.pyw` |
| .NET / C# | `dotnet/` | 4 | Stable (v0.6) — ASP.NET Core / EF Core |

**Phase v0.7+** (planned): Ruby, Rust, Java/Kotlin.

## Why specialize?

Generic rules describe **intent and reasoning** — they work cross-language. But search patterns, library names, idioms differ greatly:

- "SQL injection" in Python = `cursor.execute(f"...")`; in Go = `db.Raw(fmt.Sprintf(...))`; in PHP = `mysqli_query("$_GET[id]")`
- "Verbose error" in Go = `gin.DebugMode`; in PHP = `APP_DEBUG=true` + Ignition page (CRITICAL because leaks APP_KEY)
- "Command injection" in Go is rarer (args usually passed separately); in PHP it's common (shell_exec ubiquitous); in .NET it usually appears through `Process.Start` and `cmd.exe /c` / PowerShell wrappers.

Severity also shifts: PHP `VERBOSE-ERROR-DEBUG-MODE` is CRITICAL (Ignition leaks secrets + has past RCE CVE), while Go is HIGH (gin debug leaks routes/stack but no secret leak).

## L1–L4 reasoning still applies

Language overrides give **better patterns + examples**, NOT a shortcut around the L1–L4 data-flow analysis (see `references/data-flow-classification.md`). LLM agent must still:

1. **Grep** for sink patterns (language-specific now)
2. **Read** the full function containing the sink
3. **Trace** input back through call chain to its source (L1 request, L2 DB, L3 config, L4 constant)
4. **Verify** sanitization context (parameterized query, whitelist, escape, allowlist)

Pattern match alone produces false positives (vd: `fmt.Sprintf` in Go is fine if not in a SQL sink) and false negatives (vd: input passed through a helper function still reaches the sink).

## Contributing — add an override

For an existing language (Go or PHP):

1. Pick a rule id from `rules/generic/` not yet specialized for that language
2. Create `rules/languages/<lang>/<NN>-<name>.md` (use same numeric prefix for nav)
3. Frontmatter: copy generic, change `applies_to: <lang>` (e.g. `go`, `php`)
4. Sections to override:
   - **Intent**: lead with language-specific reasons this risk hits harder/lesser
   - **Search patterns**: replace fully with `<lang>`-specific regex (no JS examples in Go file)
   - **Examples**: ALL code blocks in target language
   - **Fix recommendation**: idiomatic `<lang>` fix
5. Keep **Cách reasoning** L1–L4 structure
6. Test: run `/vbs-scan-security` against a `<lang>` repo with the vulnerability — confirm the rule fires with the right reasoning

## Contributing — add a NEW language

To support Python / Ruby / Rust / etc.:

1. Create `rules/languages/<newlang>/`
2. Add at minimum these overrides (highest leverage):
   - `SQL-INJECTION`
   - `COMMAND-INJECTION`
   - `INSECURE-DESERIALIZATION` (Python pickle is CRITICAL!)
   - `VERBOSE-ERROR-DEBUG-MODE` (Django/Flask/FastAPI debug = leak vars)
3. Create `rules/languages/<newlang>/README.md` listing what was specialized
4. Update `references/language-detection.md`:
   - Add extension mapping
   - Add Phase table entry
5. Update `skill/SKILL.md` line "Phase 1 hiện hỗ trợ chuyên sâu" to include the new language
6. Run a real scan + 5-10 sample repos to validate before announcing the language as "stable"

## Future: rule composition

Currently override is binary (full replace). Future direction: allow language file to ONLY override specific sections (vd: only `Search patterns`) while inheriting `Intent` from generic. Not implemented yet — keep override files self-contained.

## Quality bar

- Each rule file: 80–200 lines (focused, not bloated)
- Vietnamese prose, English technical terms (consistent with generic rules)
- At least 3 CRITICAL/HIGH examples + 2 NOT-critical (safe) examples
- Cross-references to other relevant rules
- Fix recommendations are actionable (specific function/config, not "use safe practices")
