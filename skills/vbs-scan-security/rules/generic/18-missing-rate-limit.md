---
id: MISSING-RATE-LIMIT
severity_max: HIGH
applies_to: all
---

# Missing Rate Limit on Abusable Endpoints

## Intent

Endpoint mà **mỗi request đều làm hệ thống tốn tiền thật hoặc tấn công được người khác**, nhưng không có rate limit → hacker viết script gọi hàng nghìn lần. Rule này chỉ nhắm vào 4 nhóm endpoint:

1. **Gọi AI / API trả phí theo request**: LLM inference, image generation, dịch máy, geocoding, SMS gateway... → hóa đơn OpenAI/Anthropic/Twilio $50k/ngày (đã xảy ra với nhiều startup vibe code)
2. **Gửi email / SMS / push từ input người dùng**: reset password, OTP, magic link, mời bạn bè, "gửi cho tôi bản sao"... → spam nạn nhân, blacklist domain, đốt tiền SendGrid/Twilio
3. **Sinh / kiểm mã bí mật**: nhập OTP, mã khuyến mãi, mã mời, token reset → đoán được bằng vét cạn nếu không giới hạn số lần
4. **Thanh toán / tạo giao dịch**: tạo đơn, rút tiền, đổi điểm → lặp giao dịch, tạo rác, che race condition

Khác với `06-brute-force` (đăng nhập, target là mật khẩu): rule này về **chi phí, spam và vét cạn mã** ở endpoint không phải login.

## Khi nào KHÔNG tạo finding — ghi hardening note

Đây là phần quan trọng nhất của rule. Thiếu rate limit là tình trạng của gần như mọi endpoint trong code vibe, nên nếu flag hết thì báo cáo toàn nhiễu. **Không tạo finding** cho các trường hợp sau, chỉ ghi 1 dòng vào `hardening_notes[]` (không `rule_id`, không tính vào summary):

- **Endpoint đã có finding CRITICAL/HIGH của rule khác** (COMMAND-INJECTION, SSRF, SQL-INJECTION, PATH-TRAVERSAL...). Kẻ tấn công đã chiếm được server hoặc DB thì thiếu rate limit không còn ý nghĩa. Ghi "thêm rate limit" vào `fix_summary` của finding chính nếu muốn, không tạo finding thứ hai.
- **Endpoint tốn CPU/DB nhưng không tốn tiền theo request**: search / LIKE không LIMIT, export CSV, render PDF, resize ảnh, spawn process nội bộ, gọi HTTP tới host nội bộ hoặc allowlist. Đây là bài toán hiệu năng và DoS chung, không phải lỗ hổng có đường khai thác cụ thể.
- **Endpoint đã yêu cầu đăng nhập** và thuộc 4 nhóm trên nhưng chi phí mỗi request nhỏ (vd user gửi 1 email cho chính mình). Chỉ flag khi user đăng nhập vẫn gây được thiệt hại đáng kể cho người khác hoặc cho hóa đơn.
- **Không xác định được endpoint có public không** (không thấy route đăng ký, chỉ thấy hàm service).
- **Repo không có endpoint nào thuộc 4 nhóm trên**: không ghi gì, kể cả hardening note.

Nếu reasoning của mình kết luận "tác động thấp", "chỉ là DoS", "nên thêm cho chắc" → đó là hardening note, không phải finding.

## Khi nào HIGH

Endpoint public (không cần đăng nhập, hoặc đăng nhập tự do) thuộc 1 trong 4 nhóm trên, **không có** rate limiter, cooldown hay captcha:

- Gọi LLM / AI inference / image generation / API trả phí theo request
- Gửi email / SMS / push tới địa chỉ do user nhập (reset password, OTP, mời, chia sẻ)
- Kiểm mã OTP / mã khuyến mãi / mã mời có không gian nhỏ (4-8 ký tự số) mà không giới hạn số lần thử
- Tạo giao dịch tài chính hoặc rút tiền

Đây là **finding**, không phải hardening note, kể cả khi cả repo chưa có rate limiter nào và kể cả khi file khác đã có finding CRITICAL. Ví dụ điển hình: `POST /forgot-password` public gọi `sendMail` cho mọi request → HIGH.

## Khi nào MEDIUM (giảm cấp)

- Có rate limit nhưng quá lỏng so với chi phí (100 req/phút cho LLM endpoint vẫn đốt tiền)
- Rate limit chỉ per IP cho endpoint gửi mail/OTP — bypass được bằng rotating proxy, cần thêm per-account hoặc per-email
- Endpoint thuộc 4 nhóm trên nhưng đã yêu cầu đăng nhập và có thể gây thiệt hại vừa phải
- Cooldown có nhưng đặt ở frontend (disable nút), backend không kiểm

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Liệt kê endpoint thuộc 4 nhóm**: grep sink gọi LLM, mail, SMS, payment; grep handler nhận `otp`, `code`, `coupon`, `invite`, `reset`, `forgot`.
2. **Loại trước** endpoint đã có finding CRITICAL/HIGH của rule khác, endpoint chỉ tốn CPU/DB, endpoint nội bộ.
3. **Grep** rate limit middleware: `express-rate-limit`, `flask-limiter`, `fastify-rate-limit`, `django-ratelimit`, `rack-attack`, `@nestjs/throttler`, `golang.org/x/time/rate`, `ulule/limiter`, `AspNetCore.RateLimiting`, `symfony/rate-limiter`. Xem thêm cooldown tự viết (lưu `last_sent_at`, đếm trong Redis) và captcha.
4. **Read** từng endpoint còn lại: có gắn limiter / cooldown / captcha không? Limiter đặt đúng key (per user, per email) hay chỉ per IP?
5. **Ước lượng thiệt hại**: 1.000 request trong 1 phút thì ai mất gì? Không trả lời được cụ thể (tiền, spam nạn nhân, đoán ra mã) → hardening note, không phải finding.
6. **Auth không thay thế rate limit** cho nhóm 1 và 2: user đăng nhập vẫn spam được nếu chi phí mỗi request đủ lớn.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Sink thuộc 4 nhóm

```
# LLM / API trả phí
openai\.(chat|completions|images|embeddings)
anthropic\.(messages|completions)
\.generate_content|\.invoke_model|bedrock-runtime
huggingface|replicate|stability|elevenlabs|deepgram

# Email / SMS / push
sendgrid|nodemailer|smtplib|mail\(|Mail::send|wp_mail|twilio|messagebird|vonage|firebase.*messaging|expo-server-sdk
forgot|reset[-_]?password|send[-_]?otp|magic[-_]?link|invite

# Kiểm mã
verify[-_]?otp|otp\s*==|code\s*==|coupon|promo|invite[-_]?code

# Thanh toán
stripe\.(charges|paymentIntents)|withdraw|payout|transfer
```

### Rate limiter / cooldown / captcha

```
express-rate-limit|rateLimit\(
flask-limiter|@limiter\.limit
django-ratelimit|@ratelimit
fastify-rate-limit
rack-attack|Rack::Attack
@nestjs/throttler|ThrottlerGuard
golang.org/x/time/rate|ulule/limiter|tollbooth
AddRateLimiter|EnableRateLimiting|\[EnableRateLimiting
symfony/rate-limiter|RateLimiterFactory
last_sent_at|cooldown|retry_after|attempts
recaptcha|hcaptcha|turnstile
```

### Negative search

Sau khi liệt kê endpoint thuộc 4 nhóm, grep ngược lại xem route handler có gắn limiter / cooldown / captcha không. Route file không import limiter và handler không đọc `last_sent_at`/`attempts` → suspect.

## Examples

### HIGH — flag

```javascript
// Express — endpoint AI public, không giới hạn
const OpenAI = require('openai');
const openai = new OpenAI();
app.post('/api/chat', async (req, res) => {
  const result = await openai.chat.completions.create({
    model: 'gpt-4',
    messages: req.body.messages
  });
  res.json(result);
});
// Hacker script: while(true) fetch(...) → đốt $/giây
```

```python
# Flask — gửi email reset không cooldown, không captcha
@app.route('/api/reset-password', methods=['POST'])
def reset():
    email = request.json['email']
    user = User.query.filter_by(email=email).first()
    if user:
        send_reset_email(user)   # gọi liên tục cho 1 email → flood inbox nạn nhân, đốt tiền SendGrid
    return jsonify(ok=True)
```

```go
// Gin — kiểm OTP 6 số, không đếm số lần thử
func VerifyOTP(c *gin.Context) {
    var req struct{ Phone, Code string }
    c.BindJSON(&req)
    if store.Get(req.Phone) == req.Code {   // 1.000.000 khả năng, script thử hết trong vài phút
        issueSession(c, req.Phone)
    }
}
```

### KHÔNG flag — hardening note hoặc bỏ qua

```javascript
// Search LIKE không LIMIT, không rate limit → chỉ tốn DB, không tốn tiền, không tấn công được ai.
// Nếu q ghép thẳng vào SQL thì đó là SQL-INJECTION (rule 02), không phải rule này.
app.get('/search', async (req, res) => {
  const rows = await db.query('SELECT * FROM products WHERE name LIKE ?', [`%${req.query.q}%`]);
  res.json(rows);
});
// → hardening note: "thêm LIMIT + pagination cho /search"
```

```python
# /tools/ping đã có finding COMMAND-INJECTION (CRITICAL) → KHÔNG thêm MISSING-RATE-LIMIT cho cùng endpoint
@app.post("/tools/ping")
def ping():
    host = request.json.get("host", "")
    subprocess.run(f"ping -c 1 {host}", shell=True)
```

```go
// Resize ảnh / render PDF cho user đã đăng nhập → tốn CPU, không tốn tiền theo request
auth.GET("/orders/:id/invoice.pdf", a.Invoice)   // exec wkhtmltopdf
// → hardening note nếu muốn, không phải finding
```

```javascript
// Express + express-rate-limit, key per user → an toàn
const rateLimit = require('express-rate-limit');
const chatLimiter = rateLimit({
  windowMs: 60 * 1000,
  max: 20,
  keyGenerator: (req) => req.user?.id ?? req.ip,
});
app.post('/api/chat', chatLimiter, async (req, res) => { /* ... */ });
```

```python
# Flask-Limiter key theo email → an toàn
@app.route('/api/reset-password', methods=['POST'])
@limiter.limit("3 per hour", key_func=lambda: request.json['email'])
def reset(): ...
```

## Fix recommendation

1. **Cài rate-limit middleware, key theo user hoặc email** (không chỉ per IP — IP rotating dễ):
   ```javascript
   const rateLimit = require('express-rate-limit');
   app.use('/api/ai', rateLimit({
     windowMs: 60 * 1000,
     max: 10,
     keyGenerator: req => req.user?.id ?? req.ip
   }));
   ```
   ```python
   @limiter.limit("3 per hour", key_func=lambda: request.json['email'])
   ```
2. **Cooldown phía backend** cho gửi mail/OTP: lưu `last_sent_at`, từ chối nếu chưa qua 60 giây; giới hạn 3-5 lần/giờ cho mỗi email hoặc số điện thoại.
3. **Đếm số lần thử mã**: khóa OTP sau 5 lần sai, mã hết hạn sau 5-10 phút, sinh mã mới thì hủy mã cũ.
4. **Captcha** (Turnstile, hCaptcha, reCAPTCHA) cho endpoint gửi email/SMS công khai.
5. **Hard cap ngân sách** cho AI/API trả phí: đếm theo user trong DB, chặn khi vượt; alert khi chi tiêu tăng đột biến.
6. **Tier theo plan**: free 10/giờ, paid 1000/giờ.
7. **Cloudflare / WAF** rate limit ở edge cho endpoint public.

## Cross-references

- Cross-check với `06-brute-force`: đăng nhập không giới hạn là rule 06, không phải rule này
- Cross-check với `01-hardcoded-secret`: AI API key lộ + endpoint AI không giới hạn = mất tiền nhanh nhất
- Cross-check với `12-broken-access-control`: endpoint tốn tiền chỉ dành nội bộ mà vô tình public
- Cross-check với `19-race-condition`: bộ đếm rate limit không atomic thì bypass được bằng request song song
