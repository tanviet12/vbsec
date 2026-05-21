# 21 Rule bảo mật của vbsec

Tổng quan ngắn gọn từng rule với ví dụ unsafe/safe. Để đọc đầy đủ reasoning, search pattern và edge case, mở file rule tương ứng trong [`skill/rules/generic/`](../../skill/rules/generic/).

> **Ký hiệu:**
> - **Severity max** — mức cao nhất một finding của rule này có thể đạt
> - **Applies to** — `all` = mọi ngôn ngữ; có specialization riêng nếu liệt kê thêm `go`, `php`, `typescript` (gộp cả `.js/.jsx/.ts/.tsx`), `python` (`.py/.pyw`), `dotnet` (`.cs/.csproj/.sln`)

---

## Mục lục nhanh

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

API key, password DB, JWT secret, Stripe/AWS/OpenAI key viết thẳng trong code và commit lên Git. Một khi đã push lên public repo, secret coi như rò rỉ vĩnh viễn — phải xoay key, không phải chỉ xóa file.

**Unsafe (JavaScript):**
```javascript
const stripe = require('stripe')('sk_live_EXAMPLE_REAL_KEY_GOES_HERE');
```

**Safe:**
```javascript
const stripe = require('stripe')(process.env.STRIPE_SECRET_KEY);
// .env không commit, .env.example chứa key giả
```

[Đầy đủ →](../../skill/rules/generic/01-hardcoded-secret.md)

---

### Rule 2 — SQL-INJECTION

**Severity max:** CRITICAL
**Applies to:** all (+ go, php)

User input ghép trực tiếp vào câu SQL bằng concatenation hoặc f-string. Hacker chèn `' OR 1=1--` là dump cả DB. Chỉ parameterized query mới an toàn.

**Unsafe (Python):**
```python
user_id = request.args.get('id')
cursor.execute(f"SELECT * FROM users WHERE id = {user_id}")
```

**Safe:**
```python
cursor.execute("SELECT * FROM users WHERE id = %s", (user_id,))
```

[Đầy đủ →](../../skill/rules/generic/02-sql-injection.md)

---

### Rule 3 — XSS

**Severity max:** HIGH
**Applies to:** all (+ php)

Render user input ra HTML mà không escape. Hacker chèn `<script>` đánh cắp cookie / session. Framework hiện đại (React, Vue) auto-escape — chỉ nguy hiểm khi dùng `dangerouslySetInnerHTML` / `v-html` / `innerHTML`.

**Unsafe (React):**
```jsx
<div dangerouslySetInnerHTML={{__html: userBio}} />
```

**Safe:**
```jsx
<div>{userBio}</div>
// hoặc sanitize trước với DOMPurify
```

[Đầy đủ →](../../skill/rules/generic/03-xss.md)

---

### Rule 4 — IDOR

**Severity max:** HIGH
**Applies to:** all

Insecure Direct Object Reference — endpoint trả về object theo ID URL nhưng không check ownership. User A đổi `?id=1` thành `?id=2` xem được đơn hàng của user B.

**Unsafe (Express):**
```javascript
app.get('/orders/:id', async (req, res) => {
  const order = await Order.findById(req.params.id);
  res.json(order);  // KHÔNG check order.userId === req.user.id
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

[Đầy đủ →](../../skill/rules/generic/04-idor.md)

---

### Rule 5 — SLOPSQUATTING

**Severity max:** CRITICAL
**Applies to:** all

AI hallucinate tên package không tồn tại (ví dụ `requests-fast`, `react-utils-helper`). Attacker thấy code mẫu → đăng ký package đó trên npm/PyPI kèm malware. Code AI sinh ra chạy `npm install` là dính.

**Unsafe (package.json AI sinh):**
```json
"dependencies": {
  "react-form-validator-easy": "^1.0.0",  // tên này KHÔNG có thật
  "auth-helper-jwt-pro": "^2.1.0"
}
```

**Safe:**
- Verify từng dep trên npm/PyPI registry trước install
- Dùng `npm view <name>` để check tồn tại + maintainer reputation
- Pin version cụ thể, audit bằng `npm audit`

[Đầy đủ →](../../skill/rules/generic/05-slopsquatting.md)

---

### Rule 6 — BRUTE-FORCE

**Severity max:** HIGH
**Applies to:** all

Endpoint login/OTP/reset password không có rate limit hoặc account lockout. Hacker brute-force password thông dụng cả ngày không bị chặn.

**Unsafe (Flask):**
```python
@app.route('/login', methods=['POST'])
def login():
    user = User.query.filter_by(email=request.form['email']).first()
    if user and check_password(user, request.form['password']):
        return login_success()
    return 'Invalid', 401  # không limit, không lock
```

**Safe:**
```python
@app.route('/login', methods=['POST'])
@limiter.limit("5 per minute")
def login():
    # ... + lock account sau 5 lần fail liên tiếp
```

[Đầy đủ →](../../skill/rules/generic/06-brute-force.md)

---

### Rule 7 — MASS-ASSIGNMENT

**Severity max:** CRITICAL
**Applies to:** all

Endpoint update user dùng `User.update(req.body)` — attacker thêm `{"is_admin": true}` vào request body là tự promote thành admin. Phải whitelist field cho phép update.

**Unsafe (Express + Mongoose):**
```javascript
app.patch('/users/me', async (req, res) => {
  await User.findByIdAndUpdate(req.user.id, req.body);  // body có thể chứa is_admin
});
```

**Safe:**
```javascript
const { name, bio } = req.body;  // whitelist
await User.findByIdAndUpdate(req.user.id, { name, bio });
```

[Đầy đủ →](../../skill/rules/generic/07-mass-assignment.md)

---

### Rule 8 — INSECURE-DESERIALIZATION

**Severity max:** CRITICAL
**Applies to:** all (+ php)

`pickle.loads()`, `yaml.load()` (không `SafeLoader`), `unserialize()` của PHP với user input → RCE. Deserialize có thể trigger object construction → execute arbitrary code.

**Unsafe (Python):**
```python
import pickle
session_data = pickle.loads(request.cookies.get('session'))
```

**Safe:**
```python
import json
session_data = json.loads(request.cookies.get('session'))
# Hoặc dùng signed cookies (Flask: itsdangerous)
```

[Đầy đủ →](../../skill/rules/generic/08-insecure-deserialization.md)

---

### Rule 9 — SSRF

**Severity max:** HIGH
**Applies to:** all (+ go)

Server-Side Request Forgery — server fetch URL do user nhập. Hacker nhập `http://169.254.169.254/...` (AWS metadata) hoặc `http://localhost:8500/...` (internal services) để lấy credentials.

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

[Đầy đủ →](../../skill/rules/generic/09-ssrf.md)

---

### Rule 10 — PATH-TRAVERSAL

**Severity max:** HIGH
**Applies to:** all

`fs.readFile(req.params.filename)` — user truyền `../../etc/passwd` là đọc được file ngoài thư mục public.

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

[Đầy đủ →](../../skill/rules/generic/10-path-traversal.md)

---

### Rule 11 — CSRF

**Severity max:** HIGH
**Applies to:** all (+ php)

Endpoint thay đổi state (POST/PUT/DELETE) không kiểm tra CSRF token. Trang web độc tạo form auto-submit → trình duyệt user gửi request kèm cookie session → server tưởng là legit.

**Unsafe (Express):**
```javascript
app.post('/transfer', (req, res) => {
  // chỉ check session cookie, không check CSRF token
  doTransfer(req.user.id, req.body.to, req.body.amount);
});
```

**Safe:**
```javascript
const csrf = require('csurf');
app.use(csrf({ cookie: true }));
app.post('/transfer', (req, res) => {
  // csurf tự verify token trong header X-CSRF-Token
  doTransfer(...);
});
```

[Đầy đủ →](../../skill/rules/generic/11-csrf.md)

---

### Rule 12 — BROKEN-ACCESS-CONTROL

**Severity max:** CRITICAL
**Applies to:** all

Frontend ẩn nút "Delete user" cho non-admin, nhưng endpoint `DELETE /users/:id` ở backend không check role. User thường gọi thẳng API là xóa được.

**Unsafe (Flask):**
```python
@app.route('/admin/users/<id>', methods=['DELETE'])
@login_required  # chỉ check login, không check admin
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

[Đầy đủ →](../../skill/rules/generic/12-broken-access-control.md)

---

### Rule 13 — WEAK-PASSWORD-HASHING

**Severity max:** CRITICAL
**Applies to:** all (+ php)

Lưu password bằng MD5, SHA1, SHA256 thuần, hoặc plain text. Crack được trong vài phút khi DB rò rỉ. Phải dùng bcrypt / argon2 / scrypt.

**Unsafe (PHP):**
```php
$hash = md5($_POST['password']);
```

**Safe:**
```php
$hash = password_hash($_POST['password'], PASSWORD_BCRYPT);
// Verify: password_verify($input, $hash)
```

[Đầy đủ →](../../skill/rules/generic/13-weak-password-hashing.md)

---

### Rule 14 — JWT-NONE-ALGORITHM

**Severity max:** CRITICAL
**Applies to:** all

JWT verify chấp nhận `alg=none` (không signature), hoặc secret là chuỗi yếu (`'secret'`, `'changeme'`). Hacker tự forge token admin.

**Unsafe (Node.js):**
```javascript
const decoded = jwt.verify(token, 'secret');  // weak secret
// hoặc
const decoded = jwt.verify(token, key, { algorithms: ['HS256', 'none'] });
```

**Safe:**
```javascript
const decoded = jwt.verify(token, process.env.JWT_SECRET, {
  algorithms: ['HS256']  // KHÔNG có 'none'
});
// JWT_SECRET phải là random 32+ bytes
```

[Đầy đủ →](../../skill/rules/generic/14-jwt-none-algorithm.md)

---

### Rule 15 — CORS-MISCONFIG

**Severity max:** HIGH
**Applies to:** all

`Access-Control-Allow-Origin: *` kết hợp `Allow-Credentials: true` — hoặc reflect Origin header không validate. Trang web độc đọc được response API có cookie của user.

**Unsafe (Express):**
```javascript
app.use(cors({ origin: true, credentials: true }));  // reflect mọi origin
```

**Safe:**
```javascript
const ALLOWED = ['https://app.example.com'];
app.use(cors({
  origin: (origin, cb) => cb(null, ALLOWED.includes(origin)),
  credentials: true
}));
```

[Đầy đủ →](../../skill/rules/generic/15-cors-misconfig.md)

---

### Rule 16 — UNRESTRICTED-FILE-UPLOAD

**Severity max:** CRITICAL
**Applies to:** all (+ php)

Upload không validate extension/MIME, lưu vào webroot. User upload `shell.php` → access `https://site/uploads/shell.php` → RCE.

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
// Serve qua endpoint riêng, KHÔNG để trong webroot
```

[Đầy đủ →](../../skill/rules/generic/16-unrestricted-file-upload.md)

---

### Rule 17 — VERBOSE-ERROR-DEBUG-MODE

**Severity max:** HIGH
**Applies to:** all (+ go)

`DEBUG=true` ở production, stack trace lộ ra response, error chi tiết về DB query / file path. Hacker dùng thông tin này để mapping attack surface.

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

[Đầy đủ →](../../skill/rules/generic/17-verbose-error-debug-mode.md)

---

### Rule 18 — MISSING-RATE-LIMIT

**Severity max:** HIGH
**Applies to:** all

Endpoint nặng (search, export, AI inference) không có rate limit. 1 user gửi 1000 req/s là DoS.

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

[Đầy đủ →](../../skill/rules/generic/18-missing-rate-limit.md)

---

### Rule 19 — RACE-CONDITION

**Severity max:** HIGH
**Applies to:** all

Update balance / inventory không có transaction lock. 2 request rút tiền cùng lúc đều thấy balance = 100, đều trừ 100, kết quả balance = 0 (đúng ra phải fail 1 request).

**Unsafe (Go):**
```go
balance := db.GetBalance(userID)
if balance >= amount {
    db.SetBalance(userID, balance - amount)  // race window ở đây
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

[Đầy đủ →](../../skill/rules/generic/19-race-condition.md)

---

### Rule 20 — OUTDATED-DEPENDENCY

**Severity max:** HIGH
**Applies to:** all

Package có CVE đã biết (`log4j 2.14`, `lodash <4.17.21`, `express <4.17.3`). vbsec không fetch CVE real-time — flag pattern dependency cũ + đề nghị chạy `npm audit` / `pip-audit` / `govulncheck`.

**Unsafe (package.json):**
```json
"dependencies": {
  "lodash": "4.17.15",   // CVE-2021-23337 (prototype pollution)
  "express": "4.16.0"    // nhiều CVE
}
```

**Safe:**
```bash
npm audit fix
npm update
# Set up Dependabot / Renovate cho update tự động
```

[Đầy đủ →](../../skill/rules/generic/20-outdated-dependency.md)

---

### Rule 21 — COMMAND-INJECTION

**Severity max:** CRITICAL
**Applies to:** all (+ go)

`exec()`, `os.system()`, `shell=True`, `child_process.exec()` với user input → arbitrary code execution. User gửi `; rm -rf /` là server toang.

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
# Không shell=True, truyền argv list, validate filename trước
```

[Đầy đủ →](../../skill/rules/generic/21-command-injection.md)

---

## Specialization

Một số rule có override chuyên sâu cho ngôn ngữ cụ thể. Khi vbsec detect ngôn ngữ chính, nó tự load overlay:

| Ngôn ngữ | Folder | Override rule |
|---|---|---|
| Go | [`skill/rules/languages/go/`](../../skill/rules/languages/go/) | SQL-INJECTION (GORM Raw), SSRF (Colly), VERBOSE-ERROR (gin Debug), COMMAND-INJECTION (exec.Command) |
| PHP | [`skill/rules/languages/php/`](../../skill/rules/languages/php/) | SQL-INJECTION (mysqli/PDO), XSS (echo $_GET), INSECURE-DESERIALIZATION (unserialize), CSRF (Laravel), WEAK-PASSWORD-HASHING (md5), UNRESTRICTED-FILE-UPLOAD (move_uploaded_file) |
| .NET / C# | [`skill/rules/languages/dotnet/`](../../skill/rules/languages/dotnet/) | SQL-INJECTION (EF Core raw SQL), MASS-ASSIGNMENT (ASP.NET Core model binding), INSECURE-DESERIALIZATION (Newtonsoft/formatter cũ), COMMAND-INJECTION (Process.Start) |

Muốn add language khác (Ruby, Java, JS/TS, Python, Rust)? Đọc [contributing.md](contributing.md).

---

## Cập nhật rule list này

Nếu bạn thêm rule mới (22, 23...) hoặc cập nhật severity, nhớ update:

1. File này (`docs/vi/rules.md`)
2. [`docs/en/rules.md`](../en/rules.md)
3. Bảng trong [SKILL.md](../../skill/SKILL.md) Step 4
4. README.vi.md + README.en.md (section "Danh sách 21 lỗi")
