# Sub-Agent Prompts (LARGE mode)

Templates cho prompt khi main orchestrator delegate quét 1 chunk cho sub-agent. Sub-agent loại `general-purpose` (built-in Claude Code) — KHÔNG yêu cầu custom agent.

## Nguyên tắc

1. **Sub-agent KHÔNG có lang context** — Luôn yêu cầu sub-agent output **canonical English + rule ID**. Main agent sẽ translate sang `$LANG` ở step aggregate.
2. **Sub-agent KHÔNG load rule files** — Main agent embed nội dung rule (hoặc list rule ID) trực tiếp vào prompt. Sub-agent không có quyền truy cập file system bằng path tương đối từ skill, embed thẳng để tránh ambiguity.
3. **Prompt phải self-contained** — Sub-agent không thấy conversation hiện tại. Mọi context cần phải nằm trong prompt.
4. **Output file path cố định** — Sub-agent ghi findings ra `.vbsec-tmp/findings-<chunk-name-slug>.md`. Main agent đọc lại để aggregate.

## Template prompt chính

```
You are a security scanner agent for the vbsec skill (github.com/tanviet12/vbsec).
Your job: scan a chunk of files for the 21 vbsec security rules and report findings.

# Context
- Repository: {repo_path}  (= `$SCAN_ROOT`; read every file at `{repo_path}/<path>`, but report `<path>` without this prefix)
- Chunk: {chunk_name}
- Primary language: {primary_lang} (specialized rules: {yes|no})
- Files to scan ({file_count}):
{file_list}

# Rules to check
For each file, look for these 21 vulnerability categories:

1. HARDCODED-SECRET — API keys/passwords in source or committed config
2. SQL-INJECTION — untrusted input concatenated into SQL without parameterization
3. XSS — untrusted input rendered to HTML/JS without escape
4. IDOR — endpoints with id params lacking ownership/authz check
5. SLOPSQUATTING — import names that don't exist in package registry
6. BRUTE-FORCE — login/OTP/password reset endpoints without attempt limit
7. MASS-ASSIGNMENT — endpoints accepting whole object incl. privileged fields (is_admin, role)
8. INSECURE-DESERIALIZATION — pickle.loads, yaml.load, unserialize, eval on untrusted data
9. SSRF — server-side HTTP fetch with user-controlled URL/hostname
10. PATH-TRAVERSAL — file ops with user input, no `../` validation
11. CSRF — state-changing endpoint without CSRF token / SameSite cookie / Origin check
12. BROKEN-ACCESS-CONTROL — missing backend authz, role check only on frontend
13. WEAK-PASSWORD-HASHING — md5/sha1/plain text for passwords
14. JWT-NONE-ALGORITHM — alg=none allowed, weak/hardcoded JWT secret
15. CORS-MISCONFIG — Access-Control-Allow-Origin: * with credentials, regex bypass
16. UNRESTRICTED-FILE-UPLOAD — no extension+content-type validation, upload to webroot
17. VERBOSE-ERROR-DEBUG-MODE — stack traces in response, DEBUG=true, Werkzeug console
18. MISSING-RATE-LIMIT — heavy/expensive endpoints without rate limit
19. RACE-CONDITION — balance/inventory update without transaction+lock (check-then-act)
20. OUTDATED-DEPENDENCY — package.json/requirements.txt/go.mod with known-CVE versions
21. COMMAND-INJECTION — exec/system/spawn/shell=True with user input

Note: vbsec has 22 canonical rules total. Rule 22 (VULNERABLE-DEPENDENCY, live OSV.dev lookup under `--sca`) is intentionally NOT in this per-chunk list — it's a repo-wide dependency-manifest check, not something that benefits from folder-based chunking. The **main orchestrator** runs it once (SKILL.md Step 4b), then merges its findings into the aggregated result below. Do not attempt OSV lookups from within a sub-agent chunk.

# Methodology — Reasoning, not pattern-matching

For EACH potential finding:
1. READ the full function/file context (not just the matched line)
2. Trace data flow backward: where does the input come from?
   - L1 (untrusted): req.body, $_GET, request.args, msg.text, etc.
   - L2 (semi-trusted): DB read (but originated as L1)
   - L3 (trusted): hardcoded, internal computed values
   - L4 (system): env vars, file paths
3. Check sanitization on path: parameterization, escape, whitelist, type cast
4. Verdict:
   - L1 → dangerous sink, NO sanitization → REPORT
   - L3/L4 → sink → SAFE, do not report
   - L2 → check sink class (DB→DB ok; DB→HTML/shell = treat as L1)

DO NOT flag a pattern blindly. A `fmt.Sprintf` around SQL is fine if the value is L3.

# Output format

Write findings to: `.vbsec-tmp/findings-{chunk_slug}.md`

Use EN canonical (do not translate). Format:

```
# Findings — {chunk_name}

## CRITICAL
- file: path/to/file.ext
  line: 42
  rule_id: SQL-INJECTION
  issue: L1 req.body.id concatenated into raw SQL query (db.raw)
  fix: Use parameterized query with placeholder
  context: |
    function getUser(req, res) {
      const id = req.body.id;
      db.raw(`SELECT * FROM users WHERE id = ${id}`);  // <-- line 42, the dangerous line
      // attacker sends id=1' OR '1'='1 → bypass, dumps all users
    }

## HIGH
- file: ...
  line: ...
  rule_id: ...
  issue: ...
  fix: ...
  context: |
    ...

## MEDIUM
- ...

## LOW
- ...

## PASSED
- rule_id: XSS — Vue templates auto-escape, no v-html with user data
- rule_id: CSRF — Express csurf middleware applied to all state-changing routes
- ... (one line each, rule_id only if you actively verified it)

## NOT_APPLICABLE
- rule_id: JWT-NONE-ALGORITHM — no JWT usage found in this chunk
- ... (rules that don't apply to files in this chunk)

## NOT_MAPPED
- file: path/to/file.ext
  line: 42
  description: Brief description of the issue
  suggested_new_rule_id: SUGGESTED-NEW-RULE (only if you believe this is a class of vulnerability worth a new rule)
  reasoning: Why none of the 21 canonical rules fit
- ... (issues you couldn't map — usually empty; only add when truly novel)

## HARDENING_NOTES
- path/to/file.ext:12 — defense-in-depth suggestion for code that is NOT exploitable (not a finding)
- ... (usually empty; max 5)
```

# Important constraints

## Rule ID discipline (CRITICAL — read carefully)

**ONLY use the 21 canonical rule IDs listed above.** Do NOT invent new rule IDs like `INSECURE-COOKIE`, `AUTH-BYPASS`, `WEAK-CRYPTO`, `DATA-IN-URL`, `OAUTH-MISCONFIG`, `SUPPLY-CHAIN`, `INFO-DISCLOSURE`, `DATA-AT-REST`, `DEPRECATED-API`, `INSECURE-SESSION`, etc.

When you encounter a real security issue that doesn't obviously fit one of the 21 IDs, map it to the closest canonical rule using this mapping table:

| If you'd want to call it... | Use this canonical ID instead | Note in `issue` |
|---|---|---|
| INSECURE-COOKIE (httpOnly/secure/sameSite missing) | `BROKEN-ACCESS-CONTROL` | "cookie without httpOnly/secure — session theft via XSS" |
| INSECURE-SESSION (no TTL, no invalidate on logout) | `BROKEN-ACCESS-CONTROL` | "session not invalidated on logout" |
| AUTH-BYPASS (trust client header for identity) | `BROKEN-ACCESS-CONTROL` | "identity from client-supplied header" |
| DATA-IN-URL (password/token in GET URL) | `HARDCODED-SECRET` | "secret transmitted in URL — logged everywhere" |
| WEAK-CRYPTO (HMAC-SHA1, key reuse, hardcoded IV) | `WEAK-PASSWORD-HASHING` | "weak crypto primitive used: SHA1/key-reuse/..." |
| OAUTH-MISCONFIG (implicit flow, no state, etc.) | `BROKEN-ACCESS-CONTROL` | "OAuth flow misconfigured" |
| DATA-AT-REST (PCI-DSS card plaintext) | `HARDCODED-SECRET` | "sensitive data stored plaintext" |
| SUPPLY-CHAIN (curl \| sh no checksum, mutable tag) | `OUTDATED-DEPENDENCY` | "supply chain risk: unpinned install" |
| INFO-DISCLOSURE (robots.txt leak, .git exposed) | `VERBOSE-ERROR-DEBUG-MODE` | "information disclosure: ..." |
| DEPRECATED-API (apt-key, old syscall) | `OUTDATED-DEPENDENCY` | "deprecated API: ..." |

Only map when there is a **concrete exploit path** (attacker-controlled input reaches the sink, or a misconfiguration is exploitable as-is). Defense-in-depth suggestions for code that is already safe — missing lockfile, cookie flags with no XSS present, extra headers, stricter PDO options — are NOT findings: list them under `## HARDENING_NOTES` at the end of your findings file (one line each: `file:line — note`). If your own reasoning concludes the code is safe, do not emit a finding.

If you genuinely cannot map a finding to any of the 21 rules, **skip it** and mention it ONLY in the `## NOT_MAPPED` section at the end of your findings file (see format below) — main agent will decide whether to surface or propose adding a new rule in future versions.

## One finding = one primary rule_id

If a single line/block triggers multiple rules (e.g., wallet.ts has both IDOR and RACE-CONDITION), create **separate findings** — one per rule_id. Do NOT use comma-separated rule_ids. This makes counts cleaner.

Example:
```
- file: routes/wallet.ts
  line: 12
  rule_id: IDOR
  issue: req.body.UserId used as filter — attacker can drain anyone's wallet
  fix: Derive UserId from authenticated session

- file: routes/wallet.ts
  line: 12-27
  rule_id: RACE-CONDITION
  issue: Balance read-modify-write without transaction lock
  fix: Wrap in transaction with SELECT ... FOR UPDATE
```

## Other constraints

- Only report findings WITH high confidence (you traced the data flow)
- Severity is CAPPED by rule. CRITICAL rules: 01,02,05,07,08,12,13,14,16,21. HIGH max for others (03,04,06,09,10,11,15,17,18,19,20).
- You MAY downgrade severity if context reduces risk (note reason in `issue`)
- Do NOT recommend specific library versions (main agent will check OUTDATED-DEPENDENCY)
- Do NOT output JSON. Use the markdown format above. Main agent will parse.
- Keep `issue` to 1-2 lines. Put detail in `context`.
- File paths must be relative to git root.

## Context field is IMPORTANT (v0.3+)

For CRITICAL and HIGH findings, the main agent uses your `context` to generate verbose explanations for non-tech users (Why dangerous? How attacker exploits? Code before/after). Therefore your `context` should include:

- **The actual vulnerable code** (a few lines around the finding, NOT just the trigger line)
- **A marker comment** showing which line is the trigger (e.g., `// <-- line 42, the dangerous line`)
- **Optional: 1 short note** in the comment if there's a specific attacker payload (e.g., `// attacker sends id=1' OR '1'='1 → bypasses auth`)

Do NOT write the full Vietnamese/English verbose blocks yourself — main agent handles that during aggregation. Just provide rich enough `context` for paraphrasing.

# When done

Reply with the single line: `FINDINGS_WRITTEN: .vbsec-tmp/findings-{chunk_slug}.md`
```

## Template gọi sub-agent

Main agent dùng tool `Agent` (subagent_type: `general-purpose`) với prompt trên. Mỗi chunk = 1 lần spawn. Có thể spawn song song nhiều chunks trong 1 message (multiple Agent tool calls).

```
Agent({
  subagent_type: "general-purpose",
  description: "vbsec scan chunk: api/handlers",
  prompt: <template ở trên, đã fill placeholders>
})
```

## Aggregate workflow (main agent)

Sau khi tất cả sub-agents return:

1. Đọc tất cả `.vbsec-tmp/findings-*.md`
2. **Dedup**: cùng `file:line:rule_id` → giữ severity cao nhất
3. **Cross-reference**: 1 finding có liên quan đến finding khác không? (vd: HARDCODED-SECRET in .env + EXPOSED-CONFIG cùng vị trí)
4. **Merge SCA findings** (nếu `--sca`): main agent tự chạy [`../references/dependency-scan.md`](dependency-scan.md) 1 lần (không qua sub-agent), thêm findings `VULNERABLE-DEPENDENCY` vào cùng danh sách trước khi render.
5. **Translate**: Nếu `lang=vi`, dịch `issue` và `fix` sang tiếng Việt theo phrase template trong i18n file
6. **Render report** theo `output-format.md`
7. **Cleanup**: `rm -rf .vbsec-tmp/` sau khi done

## Edge cases

| Scenario | Handling |
|---|---|
| Sub-agent timeout / crash | Re-spawn cho chunk đó. Nếu fail 2 lần → note "chunk X failed to scan" trong report, tiếp tục với chunks khác. |
| Sub-agent output sai format | Try parse with tolerance. Nếu không parse được → re-spawn với prompt nhấn mạnh format. |
| Cùng finding xuất hiện 2 chunk (file ở biên) | Dedup. Đây là lý do phải có dedup step. |
| Quá nhiều findings (>500) | OK, vẫn render. Main report có note "scan này phát hiện nhiều issue — recommend ưu tiên CRITICAL trước, HIGH sau, MEDIUM/LOW làm batch". |
| Tất cả chunks PASS | Render report PASS bình thường, có note "0 findings across N chunks". |
