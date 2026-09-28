---
id: BRUTE-FORCE
severity_max: HIGH
applies_to: all
---

# Missing Rate Limit (Brute Force)

## Intent

Endpoint **login / signup / password-reset / OTP-verify / API** không có rate limit. Attacker chạy script gửi 1 triệu request thử mật khẩu hoặc OTP 6 chữ số (chỉ 1M tổ hợp = 100% trúng), **chiếm tài khoản, bypass 2FA, spam đăng ký tạo user rác, brute-force coupon code, vét cạn email hợp lệ qua signup error**.

Vibe code thường quên rate limit vì frontend chỉ cho user click 1 lần — nhưng API gọi trực tiếp được vô hạn.

## Khi nào HIGH

Chỉ HIGH khi thiếu rate limit **cộng thêm** một yếu tố làm vét cạn chắc chắn thành công hoặc hậu quả lớn:

- OTP / verify code 4-6 chữ số mà KHÔNG có lockout sau N lần fail (không gian nhỏ, vét cạn trong vài phút)
- Endpoint coupon / promo / invite code không gian nhỏ, không rate limit (vét cạn 100k mã)
- Login của **admin panel** hoặc endpoint auth mà đăng nhập xong là có quyền ghi tiền / đổi role
- Login không rate limit **và** không có MFA **và** mật khẩu lưu bằng hash yếu hoặc không có chính sách độ dài (credential stuffing thành công cao)

## Khi nào MEDIUM (mặc định)

- `/login`, `/signup`, `/forgot-password`, `/api/auth/*` của app thường KHÔNG có rate limit / lockout, không có yếu tố ở mục HIGH → **MEDIUM là mức mặc định** cho lỗi này
- Có rate limit nhưng quá lỏng (vd: 1000 req/phút cho `/login`)
- Có CAPTCHA sau N lần fail (chậm bot nhưng không chặn)

## Khi nào LOW

- Endpoint nội bộ chỉ accessible qua VPN / IP allowlist
- Có lockout theo tài khoản nhưng thiếu lockout theo IP (hoặc ngược lại)

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Grep** route handler chứa keyword `login`, `signup`, `register`, `reset`, `forgot`, `verify`, `otp`, `auth`
2. **Read** đầy đủ file + middleware stack: có dùng `express-rate-limit`, `flask-limiter`, `django-ratelimit`, `@fastify/rate-limit`, `Throttle` (NestJS), `RateLimiter` (Laravel) không?
3. **Check OTP flow**:
   - Code có bao nhiêu ký tự / digit? 4-6 digit + không lockout = đỏ
   - Có biến `attempts_count` / `failed_attempts` trong DB / Redis không?
   - Code có TTL không (vd: 5 phút)? Code sống lâu = brute dễ hơn
4. **Check global middleware**: rate limit có thể set global ở app level (vd: `app.use(rateLimiter)`) → cần đọc cả file entry
5. Reverse proxy (nginx, Cloudflare) có rate limit không? Nếu có thì OK ở mức infra.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Tìm endpoint nhạy cảm

```
(login|signin|signup|register|password-reset|forgot-password|verify-otp|verify-email|2fa)
@app\.route.*(login|signup|reset|verify)
router\.(post|put)\s*\(\s*["'].*(login|signup|reset|verify)
```

### Tìm rate-limit middleware (vắng = nghi vấn)

```
express-rate-limit|rateLimit\s*\(
flask-limiter|@limiter\.limit
django-ratelimit|@ratelimit
@fastify/rate-limit
@Throttle\s*\(                              # NestJS
RateLimiter::for\s*\(                       # Laravel
```

### Tìm OTP verification

```
(otp|code|verification_code)\s*==\s*
\.compare\s*\(\s*(otp|code|verificationCode)
verifyOTP|verify_otp|verifyCode|checkOTP
```

## Examples

### HIGH — flag

```javascript
// Express — login không rate limit
app.post("/login", async (req, res) => {
  const { username, password } = req.body;
  const user = await User.findOne({ username });
  if (await bcrypt.compare(password, user.passwordHash)) {
    return res.json({ token: sign(user) });
  }
  res.status(401).send("Wrong password");
});
```

```python
# Flask — verify OTP 6 digit không lockout, không TTL check
@app.route("/verify-otp", methods=["POST"])
def verify_otp():
    user_id = request.json["user_id"]
    code = request.json["code"]
    record = OTP.query.filter_by(user_id=user_id).first()
    if record.code == code:
        return {"verified": True}
    return {"verified": False}, 401
# Attacker brute force 000000 → 999999 trong vài phút
```

```php
<?php
// Laravel — không có throttle
Route::post('/forgot-password', [AuthController::class, 'forgot']);
```

```javascript
// Coupon brute force — không rate limit
app.get("/api/coupon/:code", async (req, res) => {
  const coupon = await Coupon.findOne({ code: req.params.code });
  if (coupon) return res.json(coupon);
  res.status(404).end();
});
```

### NOT high — không flag (hoặc downgrade)

```javascript
// Có rate limit cụ thể cho login
import rateLimit from "express-rate-limit";
const loginLimiter = rateLimit({ windowMs: 15 * 60 * 1000, max: 5 });
app.post("/login", loginLimiter, async (req, res) => { /* ... */ });
```

```python
# Flask-Limiter
from flask_limiter import Limiter
limiter = Limiter(app, key_func=get_remote_address)

@app.route("/verify-otp", methods=["POST"])
@limiter.limit("5 per minute")
def verify_otp(): ...
```

```python
# OTP có attempt count + lockout
def verify_otp(user_id, code):
    record = OTP.get(user_id)
    if record.attempts >= 5:
        raise Locked("Too many attempts")
    if record.code != code:
        record.attempts += 1
        record.save()
        raise Unauthorized()
    record.delete()
    return True
```

```php
<?php
// Laravel throttle middleware
Route::post('/forgot-password', [AuthController::class, 'forgot'])
    ->middleware('throttle:5,15');   // 5 req / 15 phút
```

## Fix recommendation

1. **Thêm rate limit middleware cho endpoint nhạy cảm**:
   ```javascript
   // Express
   import rateLimit from "express-rate-limit";
   const authLimiter = rateLimit({
     windowMs: 15 * 60 * 1000,   // 15 phút
     max: 5,                      // 5 lần / IP
     standardHeaders: true,
     message: "Quá nhiều lần thử, vui lòng đợi 15 phút",
   });
   app.use("/login", authLimiter);
   app.use("/forgot-password", authLimiter);
   ```
2. **OTP flow chuẩn**:
   - Code random 6 digit
   - TTL ngắn (5-10 phút)
   - Lockout sau 5 lần fail (Redis counter)
   - Mỗi user có **1 mã active** tại một thời điểm
   - Sau verify success → delete mã ngay
3. **Tier rate limit**:
   - Per IP: 5 req / 15 phút cho login
   - Per account: 3 req / 15 phút cho password-reset (theo email/user_id)
4. **CAPTCHA sau N fail** (Cloudflare Turnstile, hCaptcha, reCAPTCHA v3)
5. **Account lockout** (cẩn thận tránh DOS nạn nhân) — chỉ lock IP, không lock account vĩnh viễn
6. **Layer tầng infra**: Cloudflare / AWS WAF / nginx `limit_req_zone` — phòng thủ trước khi đến app

## Cross-references

- Rule `12-no-mfa-on-admin` (nếu có): admin không MFA + brute force = chiếm root
- Rule `17-verbose-error-debug-mode`: `/login` trả về "user not found" vs "wrong password" → enum user
- Rule `04-idor`: brute force ID (numeric) — cùng họ "không giới hạn"
