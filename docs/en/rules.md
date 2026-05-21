# The 21 vbsec Security Rules

Compact overview of each rule with unsafe/safe examples. For the full reasoning, search patterns, and edge cases, open the corresponding rule file under [`skill/rules/generic/`](../../skill/rules/generic/).

> **Conventions:**
> - **Severity max** — the highest severity a finding of this rule can receive
> - **Applies to** — `all` = every language; specialized overlays listed if `go`, `php`, `typescript` (includes `.js/.jsx/.ts/.tsx`), `python` (`.py/.pyw`), `dotnet` (`.cs/.csproj/.sln`)

---

## Quick index

| # | Rule ID | Severity max | Specialization |
|---|---|---|---|
| 1 | [HARDCODED-SECRET](#rule-1--hardcoded-secret) | CRITICAL | — |
| 2 | [SQL-INJECTION](#rule-2--sql-injection) | CRITICAL | go, php, typescript, python, dotnet |
| 3 | [XSS](#rule-3--xss) | HIGH | typescript, python |
| 4 | [IDOR](#rule-4--idor) | HIGH | — |
| 5 | [SLOPSQUATTING](#rule-5--slopsquatting) | CRITICAL | — |
| 6 | [BRUTE-FORCE](#rule-6--brute-force) | HIGH | — |
| 7 | [MASS-ASSIGNMENT](#rule-7--mass-assignment) | CRITICAL | typescript, python, dotnet |
| 8 | [INSECURE-DESERIALIZATION](#rule-8--insecure-deserialization) | CRITICAL | go, php, typescript, python, dotnet |
| 9 | [SSRF](#rule-9--ssrf) | HIGH | go, typescript, python |
| 10 | [PATH-TRAVERSAL](#rule-10--path-traversal) | HIGH | — |
| 11 | [CSRF](#rule-11--csrf) | HIGH | php, typescript, python |
| 12 | [BROKEN-ACCESS-CONTROL](#rule-12--broken-access-control) | CRITICAL | — |
| 13 | [WEAK-PASSWORD-HASHING](#rule-13--weak-password-hashing) | CRITICAL | — |
| 14 | [JWT-NONE-ALGORITHM](#rule-14--jwt-none-algorithm) | CRITICAL | typescript, python |
| 15 | [CORS-MISCONFIG](#rule-15--cors-misconfig) | HIGH | typescript, python |
| 16 | [UNRESTRICTED-FILE-UPLOAD](#rule-16--unrestricted-file-upload) | CRITICAL | — |
| 17 | [VERBOSE-ERROR-DEBUG-MODE](#rule-17--verbose-error-debug-mode) | HIGH | go, php, typescript, python |
| 18 | [MISSING-RATE-LIMIT](#rule-18--missing-rate-limit) | HIGH | — |
| 19 | [RACE-CONDITION](#rule-19--race-condition) | HIGH | — |
| 20 | [OUTDATED-DEPENDENCY](#rule-20--outdated-dependency) | HIGH | — |
| 21 | [COMMAND-INJECTION](#rule-21--command-injection) | CRITICAL | go, php, typescript, python, dotnet |

---

### Rule 1 — HARDCODED-SECRET

**Severity max:** CRITICAL
**Applies to:** all

API keys, DB passwords, JWT secrets, Stripe/AWS/OpenAI keys committed in source. Once pushed to a public repo, the secret is permanently leaked — you must rotate, not just delete.

**Unsafe (JavaScript):**
```javascript
const stripe = require('stripe')('sk_live_EXAMPLE_REAL_KEY_GOES_HERE');
```

**Safe:**
```javascript
const stripe = require('stripe')(process.env.STRIPE_SECRET_KEY);
// .env is gitignored; .env.example holds placeholder keys
```

[Full reasoning →](../../skill/rules/generic/01-hardcoded-secret.md)

---

### Rule 2 — SQL-INJECTION

**Severity max:** CRITICAL
**Applies to:** all (+ go, php)

User input concatenated into an SQL string via `+` or f-strings. Attacker injects `' OR 1=1--` and dumps the whole DB. Only parameterized queries are safe.

**Unsafe (Python):**
```python
user_id = request.args.get('id')
cursor.execute(f"SELECT * FROM users WHERE id = {user_id}")
```

**Safe:**
```python
cursor.execute("SELECT * FROM users WHERE id = %s", (user_id,))
```

[Full reasoning →](../../skill/rules/generic/02-sql-injection.md)

---

### Rule 3 — XSS

**Severity max:** HIGH
**Applies to:** all (+ php)

Rendering user input into HTML without escaping. Attacker injects `<script>` to steal cookies / sessions. Modern frameworks (React, Vue) auto-escape — danger appears with `dangerouslySetInnerHTML` / `v-html` / `innerHTML`.

**Unsafe (React):**
```jsx
<div dangerouslySetInnerHTML={{__html: userBio}} />
```

**Safe:**
```jsx
<div>{userBio}</div>
// Or sanitize first with DOMPurify
```

[Full reasoning →](../../skill/rules/generic/03-xss.md)

---

### Rule 4 — IDOR

**Severity max:** HIGH
**Applies to:** all

Insecure Direct Object Reference — endpoint returns an object by URL ID without checking ownership. User A swaps `?id=1` to `?id=2` and reads user B's order.

**Unsafe (Express):**
```javascript
app.get('/orders/:id', async (req, res) => {
  const order = await Order.findById(req.params.id);
  res.json(order);  // Doesn't check order.userId === req.user.id
});
```

**Safe:**
```javascript
app.get('/orders/:id', async (req, res) => {
  const order = await Order.findOne({_id: req.params.id, userId: req.user.id});
  if (!order) return res.status(404).end();
  res.json(order);
});
```

[Full reasoning →](../../skill/rules/generic/04-idor.md)

---

### Rule 5 — SLOPSQUATTING

**Severity max:** CRITICAL
**Applies to:** all

AI hallucinates non-existent package names (e.g., `requests-fast`, `react-utils-helper`). Attackers spot the pattern and publish a malicious package with that exact name. AI-generated code that runs `npm install` is compromised.

**Unsafe (AI-generated package.json):**
```json
"dependencies": {
  "react-form-validator-easy": "^1.0.0",  // This name is fabricated
  "auth-helper-jwt-pro": "^2.1.0"
}
```

**Safe:**
- Verify each dep on the npm/PyPI registry before installing
- Use `npm view <name>` to check existence + maintainer reputation
- Pin versions, audit with `npm audit`

[Full reasoning →](../../skill/rules/generic/05-slopsquatting.md)

---

### Rule 6 — BRUTE-FORCE

**Severity max:** HIGH
**Applies to:** all

Login/OTP/password-reset endpoints with no rate limit and no account lockout. Attacker brute-forces common passwords all day unchecked.

**Unsafe (Flask):**
```python
@app.route('/login', methods=['POST'])
def login():
    user = User.query.filter_by(email=request.form['email']).first()
    if user and check_password(user, request.form['password']):
        return login_success()
    return 'Invalid', 401  # No limit, no lockout
```

**Safe:**
```python
@app.route('/login', methods=['POST'])
@limiter.limit("5 per minute")
def login():
    # ... + lock account after 5 consecutive failures
```

[Full reasoning →](../../skill/rules/generic/06-brute-force.md)

---

### Rule 7 — MASS-ASSIGNMENT

**Severity max:** CRITICAL
**Applies to:** all

User update endpoint takes `User.update(req.body)` blindly. Attacker adds `{"is_admin": true}` to the request body and self-promotes. Always whitelist updatable fields.

**Unsafe (Express + Mongoose):**
```javascript
app.patch('/users/me', async (req, res) => {
  await User.findByIdAndUpdate(req.user.id, req.body);  // body may contain is_admin
});
```

**Safe:**
```javascript
const { name, bio } = req.body;  // whitelist
await User.findByIdAndUpdate(req.user.id, { name, bio });
```

[Full reasoning →](../../skill/rules/generic/07-mass-assignment.md)

---

### Rule 8 — INSECURE-DESERIALIZATION

**Severity max:** CRITICAL
**Applies to:** all (+ php)

`pickle.loads()`, `yaml.load()` without `SafeLoader`, PHP's `unserialize()` on user input → RCE. Deserialization can trigger object construction → arbitrary code execution.

**Unsafe (Python):**
```python
import pickle
session_data = pickle.loads(request.cookies.get('session'))
```

**Safe:**
```python
import json
session_data = json.loads(request.cookies.get('session'))
# Or use signed cookies (Flask: itsdangerous)
```

[Full reasoning →](../../skill/rules/generic/08-insecure-deserialization.md)

---

### Rule 9 — SSRF

**Severity max:** HIGH
**Applies to:** all (+ go)

Server-Side Request Forgery — server fetches a user-supplied URL. Attacker submits `http://169.254.169.254/...` (AWS metadata) or `http://localhost:8500/...` (internal services) to steal credentials.

**Unsafe (Node.js):**
```javascript
app.post('/preview', async (req, res) => {
  const html = await fetch(req.body.url).then(r => r.text());
  res.send(html);
});
```

**Safe:**
```javascript
const ALLOWED_HOSTS = ['example.com', 'cdn.example.com'];
const url = new URL(req.body.url);
if (!ALLOWED_HOSTS.includes(url.hostname)) {
  return res.status(400).send('Host not allowed');
}
// + block private IP ranges (169.254.*, 127.*, 10.*)
```

[Full reasoning →](../../skill/rules/generic/09-ssrf.md)

---

### Rule 10 — PATH-TRAVERSAL

**Severity max:** HIGH
**Applies to:** all

`fs.readFile(req.params.filename)` — user submits `../../etc/passwd` and reads files outside the public folder.

**Unsafe (Express):**
```javascript
app.get('/files/:name', (req, res) => {
  res.sendFile(`/var/www/uploads/${req.params.name}`);
});
```

**Safe:**
```javascript
const path = require('path');
const requested = path.resolve('/var/www/uploads', req.params.name);
if (!requested.startsWith('/var/www/uploads/')) {
  return res.status(403).end();
}
res.sendFile(requested);
```

[Full reasoning →](../../skill/rules/generic/10-path-traversal.md)

---

### Rule 11 — CSRF

**Severity max:** HIGH
**Applies to:** all (+ php)

State-changing endpoint (POST/PUT/DELETE) without a CSRF token check. A malicious site auto-submits a form — the user's browser sends the request with their session cookie → server thinks it's legitimate.

**Unsafe (Express):**
```javascript
app.post('/transfer', (req, res) => {
  // Only checks session cookie, no CSRF token
  doTransfer(req.user.id, req.body.to, req.body.amount);
});
```

**Safe:**
```javascript
const csrf = require('csurf');
app.use(csrf({ cookie: true }));
app.post('/transfer', (req, res) => {
  // csurf auto-verifies the X-CSRF-Token header
  doTransfer(...);
});
```

[Full reasoning →](../../skill/rules/generic/11-csrf.md)

---

### Rule 12 — BROKEN-ACCESS-CONTROL

**Severity max:** CRITICAL
**Applies to:** all

Frontend hides the "Delete user" button for non-admins, but the `DELETE /users/:id` endpoint has no role check. Regular users can call the API directly and delete.

**Unsafe (Flask):**
```python
@app.route('/admin/users/<id>', methods=['DELETE'])
@login_required  # only checks login, not admin
def delete_user(id):
    User.query.get(id).delete()
```

**Safe:**
```python
@app.route('/admin/users/<id>', methods=['DELETE'])
@login_required
@admin_required
def delete_user(id):
    User.query.get(id).delete()
```

[Full reasoning →](../../skill/rules/generic/12-broken-access-control.md)

---

### Rule 13 — WEAK-PASSWORD-HASHING

**Severity max:** CRITICAL
**Applies to:** all (+ php)

Storing passwords with MD5, SHA1, plain SHA256, or plain text. Crackable in minutes if the DB leaks. Use bcrypt / argon2 / scrypt.

**Unsafe (PHP):**
```php
$hash = md5($_POST['password']);
```

**Safe:**
```php
$hash = password_hash($_POST['password'], PASSWORD_BCRYPT);
// Verify with: password_verify($input, $hash)
```

[Full reasoning →](../../skill/rules/generic/13-weak-password-hashing.md)

---

### Rule 14 — JWT-NONE-ALGORITHM

**Severity max:** CRITICAL
**Applies to:** all

JWT verifier accepts `alg=none` (no signature) or uses a weak secret (`'secret'`, `'changeme'`). Attacker forges admin tokens at will.

**Unsafe (Node.js):**
```javascript
const decoded = jwt.verify(token, 'secret');  // weak secret
// Or
const decoded = jwt.verify(token, key, { algorithms: ['HS256', 'none'] });
```

**Safe:**
```javascript
const decoded = jwt.verify(token, process.env.JWT_SECRET, {
  algorithms: ['HS256']  // No 'none'
});
// JWT_SECRET must be 32+ bytes of randomness
```

[Full reasoning →](../../skill/rules/generic/14-jwt-none-algorithm.md)

---

### Rule 15 — CORS-MISCONFIG

**Severity max:** HIGH
**Applies to:** all

`Access-Control-Allow-Origin: *` combined with `Allow-Credentials: true`, or reflecting the Origin header without validation. A malicious site can read your API response complete with the victim's cookies.

**Unsafe (Express):**
```javascript
app.use(cors({ origin: true, credentials: true }));  // reflects every origin
```

**Safe:**
```javascript
const ALLOWED = ['https://app.example.com'];
app.use(cors({
  origin: (origin, cb) => cb(null, ALLOWED.includes(origin)),
  credentials: true
}));
```

[Full reasoning →](../../skill/rules/generic/15-cors-misconfig.md)

---

### Rule 16 — UNRESTRICTED-FILE-UPLOAD

**Severity max:** CRITICAL
**Applies to:** all (+ php)

Upload endpoint doesn't validate extension/MIME and saves into webroot. Attacker uploads `shell.php` → hits `https://site/uploads/shell.php` → RCE.

**Unsafe (PHP):**
```php
move_uploaded_file($_FILES['file']['tmp_name'], 'uploads/' . $_FILES['file']['name']);
```

**Safe:**
```php
$ext = strtolower(pathinfo($_FILES['file']['name'], PATHINFO_EXTENSION));
if (!in_array($ext, ['jpg', 'png', 'pdf'])) die('Bad type');
$newName = bin2hex(random_bytes(16)) . '.' . $ext;
move_uploaded_file($_FILES['file']['tmp_name'], '/var/uploads-private/' . $newName);
// Serve via a dedicated endpoint; never store directly in webroot
```

[Full reasoning →](../../skill/rules/generic/16-unrestricted-file-upload.md)

---

### Rule 17 — VERBOSE-ERROR-DEBUG-MODE

**Severity max:** HIGH
**Applies to:** all (+ go)

`DEBUG=true` in production, stack traces leaking to responses, detailed errors revealing DB query / file paths. Attackers map attack surface from this info.

**Unsafe (Django settings.py):**
```python
DEBUG = True
ALLOWED_HOSTS = ['*']
```

**Safe:**
```python
DEBUG = os.environ.get('DEBUG', 'False') == 'True'
ALLOWED_HOSTS = os.environ.get('ALLOWED_HOSTS', '').split(',')
# Production: DEBUG=False, ALLOWED_HOSTS=example.com
```

[Full reasoning →](../../skill/rules/generic/17-verbose-error-debug-mode.md)

---

### Rule 18 — MISSING-RATE-LIMIT

**Severity max:** HIGH
**Applies to:** all

Heavy endpoints (search, export, AI inference) with no rate limit. A single user firing 1000 req/s causes DoS.

**Unsafe (Express):**
```javascript
app.get('/api/search', async (req, res) => {
  const results = await heavyDbQuery(req.query.q);
  res.json(results);
});
```

**Safe:**
```javascript
const rateLimit = require('express-rate-limit');
const searchLimit = rateLimit({ windowMs: 60_000, max: 30 });
app.get('/api/search', searchLimit, async (req, res) => { ... });
```

[Full reasoning →](../../skill/rules/generic/18-missing-rate-limit.md)

---

### Rule 19 — RACE-CONDITION

**Severity max:** HIGH
**Applies to:** all

Balance/inventory updates without a transaction lock. Two concurrent withdraw requests both see balance = 100, both subtract 100, balance ends at 0 (one should have failed).

**Unsafe (Go):**
```go
balance := db.GetBalance(userID)
if balance >= amount {
    db.SetBalance(userID, balance - amount)  // race window here
}
```

**Safe:**
```go
tx := db.Begin()
defer tx.Rollback()
var balance int
tx.Raw("SELECT balance FROM accounts WHERE id=? FOR UPDATE", userID).Scan(&balance)
if balance < amount { return errors.New("insufficient") }
tx.Exec("UPDATE accounts SET balance=balance-? WHERE id=?", amount, userID)
tx.Commit()
```

[Full reasoning →](../../skill/rules/generic/19-race-condition.md)

---

### Rule 20 — OUTDATED-DEPENDENCY

**Severity max:** HIGH
**Applies to:** all

Packages with known CVEs (`log4j 2.14`, `lodash <4.17.21`, `express <4.17.3`). vbsec doesn't fetch CVE data in real-time — it flags old dependency patterns and recommends `npm audit` / `pip-audit` / `govulncheck`.

**Unsafe (package.json):**
```json
"dependencies": {
  "lodash": "4.17.15",   // CVE-2021-23337 (prototype pollution)
  "express": "4.16.0"    // multiple CVEs
}
```

**Safe:**
```bash
npm audit fix
npm update
# Set up Dependabot / Renovate for ongoing updates
```

[Full reasoning →](../../skill/rules/generic/20-outdated-dependency.md)

---

### Rule 21 — COMMAND-INJECTION

**Severity max:** CRITICAL
**Applies to:** all (+ go)

`exec()`, `os.system()`, `shell=True`, `child_process.exec()` with user input → arbitrary code execution. User sends `; rm -rf /` and your server's gone.

**Unsafe (Python):**
```python
import os
filename = request.args.get('file')
os.system(f"convert {filename} output.png")
```

**Safe:**
```python
import subprocess
subprocess.run(['convert', filename, 'output.png'], check=True)
# No shell=True, argv as a list, validate filename first
```

[Full reasoning →](../../skill/rules/generic/21-command-injection.md)

---

## Specializations

Some rules have language-specific overrides that catch idioms more accurately. When vbsec detects the primary language, it loads the matching overlay:

| Language | Folder | Overridden rules |
|---|---|---|
| Go | [`skill/rules/languages/go/`](../../skill/rules/languages/go/) | SQL-INJECTION (GORM Raw), SSRF (Colly), VERBOSE-ERROR (gin Debug), COMMAND-INJECTION (exec.Command) |
| PHP | [`skill/rules/languages/php/`](../../skill/rules/languages/php/) | SQL-INJECTION (mysqli/PDO), XSS (echo $_GET), INSECURE-DESERIALIZATION (unserialize), CSRF (Laravel), WEAK-PASSWORD-HASHING (md5), UNRESTRICTED-FILE-UPLOAD (move_uploaded_file) |
| .NET / C# | [`skill/rules/languages/dotnet/`](../../skill/rules/languages/dotnet/) | SQL-INJECTION (EF Core raw SQL), MASS-ASSIGNMENT (ASP.NET Core model binding), INSECURE-DESERIALIZATION (Newtonsoft/legacy formatters), COMMAND-INJECTION (Process.Start) |

Want to add another language (Ruby, Java, JS/TS, Python, Rust)? See [contributing.md](contributing.md).

---

## Updating this rule list

If you add a new rule (22, 23...) or change a severity, remember to update:

1. This file (`docs/en/rules.md`)
2. [`docs/vi/rules.md`](../vi/rules.md)
3. The table in [SKILL.md](../../skill/SKILL.md) Step 4
4. README.vi.md + README.en.md (the "21 vulnerabilities" section)
