---
id: CSRF
severity_max: HIGH
applies_to: all
---

# Cross-Site Request Forgery (CSRF)

## Intent

Browser tự động gửi cookie session cho mọi request đến domain đó. Nếu app dùng **cookie auth** và state-changing endpoint (POST/PUT/DELETE) **không có CSRF token** hoặc cookie không có `SameSite=Lax/Strict`, attacker tạo trang `evil.com` chứa form auto-submit hoặc `<img src="https://victim.com/transfer?to=attacker&amount=1000">`. Nạn nhân click link → request gửi đi với cookie nạn nhân → **đổi mật khẩu, chuyển tiền, xóa tài khoản** trong khi nạn nhân không biết.

Lưu ý quan trọng: **JWT trong header `Authorization`** thì miễn nhiễm CSRF (browser không tự gửi header này). Chỉ cookie-based auth mới dính.

## Khi nào HIGH

Cần đủ **cả hai** điều kiện:

1. Auth dùng **session cookie** (Laravel session, Django session, Express `express-session`, PHP `session_start`, Rails cookie store) và cookie KHÔNG có `SameSite=Lax/Strict` hiệu lực
2. Endpoint state-changing KHÔNG có CSRF token / Origin check **và** hành động nhạy cảm: chuyển tiền, đổi email / password / số điện thoại, xóa tài khoản, đổi role, thêm API key, đổi địa chỉ nhận hàng

## Khi nào MEDIUM (mặc định)

- Đủ điều kiện 1 nhưng endpoint chỉ đổi dữ liệu ít nhạy cảm: display name, avatar, cài đặt hiển thị, bình luận, đăng ký nhận tin → **MEDIUM là mức mặc định** cho CSRF
- Có check `Origin` / `Referer` header nhưng thiếu ở một số endpoint
- App là **SPA + JWT trong header** nhưng còn cookie hybrid cho vài route

## Khi nào LOW

- Cookie set `SameSite=Strict` (chặn hầu hết CSRF nhưng vẫn có gap: subdomain takeover)
- Hành động chỉ tạo dữ liệu rác cho chính nạn nhân (upload avatar, thêm bình luận) và có thể hoàn tác

## Khi nào KHÔNG flag

- API hoàn toàn JWT-in-header, KHÔNG dùng cookie cho auth
- Endpoint chỉ GET (read-only, không thay đổi state) — GET nên idempotent
- Framework auto bật CSRF protection và developer không tắt (Django, Rails, Laravel mặc định ON)

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Identify auth mechanism**:
   - Tìm code set/read cookie session: `req.session`, `request.session`, `session_start()`, `cookie-parser`
   - Tìm JWT verify từ header: `req.headers.authorization`, `Bearer `
2. **Identify state-changing routes**: POST/PUT/PATCH/DELETE
3. **Verify CSRF protection**:
   - Django: `{% csrf_token %}` trong form, `csrfmiddlewaretoken` field, `CsrfViewMiddleware` enabled
   - Rails: `protect_from_forgery with: :exception` (default ON)
   - Laravel: `VerifyCsrfToken` middleware, `@csrf` blade directive
   - Express: `csurf` / `csrf-csrf` middleware, hoặc check `Origin`
   - ASP.NET: `[ValidateAntiForgeryToken]`
4. **Check cookie SameSite**:
   ```
   res.cookie('sid', token, { sameSite: 'lax', httpOnly: true, secure: true })
   ```
5. Nếu app SPA gọi API: kiểm tra `Origin` header match domain mong đợi, hoặc dùng custom header (vd: `X-Requested-With`) buộc CORS preflight.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Tìm dùng cookie session

```
express-session|cookie-session
request\.session|req\.session
SESSION_COOKIE_NAME
session_start\s*\(
```

### Tìm vắng CSRF middleware (Express)

```
require\s*\(\s*["']csurf["']
csrf\s*\(\s*\)
@fastify/csrf-protection
```

### Django (cảnh báo nếu tắt)

```
@csrf_exempt
csrf_exempt
CSRF_TRUSTED_ORIGINS                         # check setting
MIDDLEWARE.*CsrfViewMiddleware              # phải có
```

### Rails / Laravel

```
skip_before_action\s*:verify_authenticity_token
\\$except\s*=\s*\[                          # Laravel VerifyCsrfToken exception
```

### Cookie attributes thiếu

```
\.cookie\s*\(.*\)\s*$                       # check trong block có sameSite + httpOnly không
set-cookie.*samesite=none
```

## Examples

### HIGH — flag

```javascript
// Express dùng cookie session — KHÔNG có csurf
const session = require("express-session");
app.use(session({ secret: process.env.SESSION_SECRET, cookie: { httpOnly: true } }));
// KHÔNG có app.use(csrf())
app.post("/api/transfer", requireAuth, (req, res) => {
  transferMoney(req.session.userId, req.body.to, req.body.amount);
  res.json({ ok: true });
});
// Attacker tạo evil.com với <form action="https://victim.com/api/transfer" method="POST">...
```

```python
# Django — VIEW tắt CSRF cho convenience
from django.views.decorators.csrf import csrf_exempt

@csrf_exempt
def change_email(request):
    request.user.email = request.POST["email"]
    request.user.save()
    return JsonResponse({"ok": True})
```

```php
<?php
// Vanilla PHP, không có token
session_start();
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    update_password($_SESSION['user_id'], $_POST['new_password']);
}
```

```javascript
// Cookie set SameSite=None mà không có CSRF token
res.cookie("sid", token, { sameSite: "none", secure: true });
// → cookie gửi cross-site freely, không có gì chặn CSRF
```

### NOT high — không flag (hoặc downgrade)

```javascript
// JWT in header — implicit CSRF-safe
app.post("/api/transfer", (req, res) => {
  const token = req.headers.authorization?.replace("Bearer ", "");
  const user = jwt.verify(token, secret);
  // browser KHÔNG tự gửi Authorization header cross-site
});
```

```javascript
// Express + csurf
import csrf from "csurf";
app.use(csrf({ cookie: true }));
app.post("/api/transfer", (req, res) => { ... });
// Frontend phải gửi token trong header X-CSRF-Token
```

```python
# Django default — CsrfViewMiddleware ON
MIDDLEWARE = [
    ...
    "django.middleware.csrf.CsrfViewMiddleware",
]
# Form có {% csrf_token %} — an toàn
```

```javascript
// Cookie SameSite=Strict
res.cookie("sid", token, { sameSite: "strict", httpOnly: true, secure: true });
```

## Fix recommendation

1. **Dùng JWT-in-header cho API thuần SPA** — đơn giản nhất, tự miễn nhiễm CSRF (nhưng cẩn thận XSS đọc localStorage → cross-check rule `03-xss`).
2. **Nếu phải dùng cookie session**: bật CSRF middleware:
   - Express:
     ```javascript
     import { doubleCsrf } from "csrf-csrf";
     const { doubleCsrfProtection, generateToken } = doubleCsrf({ getSecret: () => process.env.CSRF_SECRET });
     app.use(doubleCsrfProtection);
     ```
   - Django: KHÔNG `@csrf_exempt` cho POST nhạy cảm. Form luôn `{% csrf_token %}`. Fetch API: gửi `X-CSRFToken` header.
   - Rails: giữ `protect_from_forgery with: :exception` (default).
   - Laravel: giữ `VerifyCsrfToken` middleware. Chỉ exempt webhook endpoint cần thiết.
3. **Set cookie attributes**:
   ```javascript
   res.cookie("sid", token, {
     httpOnly: true,
     secure: true,
     sameSite: "lax",   // hoặc "strict" cho endpoint nhạy cảm
     maxAge: 7 * 24 * 3600 * 1000,
   });
   ```
4. **Double Submit Cookie** (pattern phổ biến cho SPA):
   - Server set cookie `csrf_token=<random>`
   - JS đọc cookie, gửi giá trị trong header `X-CSRF-Token`
   - Server so sánh header với cookie
5. **Verify `Origin` / `Referer` header** ở state-changing endpoint:
   ```javascript
   const origin = req.headers.origin || req.headers.referer;
   if (!origin || !origin.startsWith("https://myapp.com")) return res.status(403).end();
   ```
6. **Custom header buộc preflight**: SPA gửi `X-Requested-With: XMLHttpRequest` → CORS preflight → cross-site form không gửi được header này → chặn được CSRF.

## Cross-references

- Rule `03-xss`: XSS bypass CSRF (đọc token từ DOM rồi tự gửi)
- Rule `04-idor`: DELETE/PUT IDOR + thiếu CSRF = exploit qua link click
- Rule `06-brute-force`: cookie session brute force kết hợp CSRF
