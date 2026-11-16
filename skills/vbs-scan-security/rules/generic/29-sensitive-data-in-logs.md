---
id: SENSITIVE-DATA-IN-LOGS
severity_max: HIGH
applies_to: all
cwe: [CWE-532, CWE-117, CWE-779]
owasp: [A09:2021]
---

# Sensitive Data in Logs / Audit Trail

## Intent

Log và bảng nhật ký có nhiều người đọc hơn (vận hành, hỗ trợ, dịch vụ thu log), giữ lâu hơn, hiếm khi mã hoá. Mật khẩu, token, OTP, lương, mã số thuế, số thẻ lọt vào đó là lộ ra ngoài vòng phân quyền của app. Nhóm thứ hai: nhật ký kiểm toán **không đáng tin** — app tự sửa được, bị chèn dòng giả, nối sự kiện bằng id do client đặt.

Mẫu hay gặp: middleware / dispatcher chung ghi nguyên tham số **mọi** lượt gọi, chỉ lọc khoá tên giống secret. Endpoint "chỉ ghi" dữ liệu nhạy không trả gì ra, nhưng đầu vào của nó nằm nguyên trong nhật ký.

## Khi nào HIGH

- Ghi dạng chữ: mật khẩu, token phiên / refresh, API key, OTP, mã reset, số thẻ đầy đủ, CVV. Gồm log body của endpoint đăng nhập / reset, dump header `Authorization` / `Cookie`, URL có token.
- Logger chung ghi nguyên body / tham số, trong phạm vi có endpoint nhận lương, mã số thuế, số giấy tờ, sao kê, hồ sơ y tế; không có che, hoặc chỉ lọc theo tên khoá kiểu `password|token`.
- Trường được mã hoá ở bảng nghiệp vụ nhưng có bản chữ ở bảng nhật ký, hàng đợi job, bảng thử lại webhook.
- Nhật ký hoạt động mà vai thấp đọc được chứa số tiền / lý do chữ tự do của khu vai đó không được xem.

## Khi nào MEDIUM

- Lỗi bên thứ ba (SMTP, cổng thanh toán, API ngoài) log nguyên văn, trong đó có email, token, body phản hồi.
- PII thường (email, số điện thoại, địa chỉ) ghi ở mức info trên đường chạy thường xuyên; giá trị lồng (map / struct / object) ghi nguyên vào nhật ký.
- **Log injection**: giá trị do client gửi (username, User-Agent, tham số) nối vào dòng log dạng chữ mà không bỏ `\r\n` → giả được dòng log.
- Request id / trace id lấy từ header client rồi dùng làm khoá **nối sự kiện kiểm toán**: kẻ tấn công đặt id trùng để trộn hoặc che hành vi.
- Nhật ký tuyên bố bất biến nhưng repo có bằng chứng app sửa được: code `UPDATE` / `DELETE` trên bảng nhật ký, hoặc migration / compose cấp cho user DB của app toàn quyền trên bảng đó.
- Sự kiện người chưa đăng nhập kích hoạt được (đăng nhập sai, quên mật khẩu) ghi nhật ký không trần → làm chìm sự kiện thật.

## Khi nào KHÔNG tạo finding

- Ghi id, mã lỗi, kích thước, số phần tử, 4 số cuối thẻ, token đã băm hoặc chỉ tiền tố ngắn.
- Có lớp che trước logger phủ đúng trường nhạy (đọc hàm che; lọc theo tên khoá vẫn flag nếu trường nhạy không có tên giống secret).
- Tên nghe nhạy nhưng giá trị không phải bí mật: `tokenCount`, `passwordPolicy`.
- Code test, seed, script dev cục bộ.
- Secret viết cứng trong mã (kể cả trong lời gọi log) → HARDCODED-SECRET.
- Dữ liệu nhạy / stack trace trả ra **response** → VERBOSE-ERROR-DEBUG-MODE.
- Logger có cấu trúc (JSON: `slog`, `zap`, `pino`, `structlog`) tự escape xuống dòng → không flag log injection.
- Request id của client chỉ in ra để debug, không làm khoá kiểm toán.
- "Thiếu log", "chưa có audit trail" → hardening note.
- Không có bằng chứng về quyền DB trong repo → không đoán "user DB sửa được nhật ký".

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Tìm điểm ghi chung** trước: middleware log request, dispatcher RPC / tool, interceptor HTTP client, handler lỗi toàn cục, hàm `audit()`.
2. **Xác định cái gì được ghi**: nguyên body? nguyên struct? header? Có hàm che không, che theo allowlist hay theo tên khoá?
3. **Đối chiếu endpoint nhạy** sau điểm ghi đó: đăng nhập, reset, 2FA, lương, thuế, thanh toán. Bộ lọc có bắt được tên trường nhạy không (`amount`, `tax_code`, `code` không giống secret).
4. **Log lỗi**: `log(err)` với lỗi từ adapter ngoài. Lỗi đó chứa gì (địa chỉ nhận, URL có token, body)?
5. **Ai đọc được**: log file / stdout hay bảng nhật ký có màn xem (vai nào).
6. **Độ tin của nhật ký**: grep `UPDATE` / `DELETE` trên bảng nhật ký; đọc GRANT trong migration; khoá nối sự kiện lấy từ đâu.
7. **Log injection**: chỉ khi logger ghi dòng chữ tự do và giá trị đến từ request.
8. Finding nêu: trường nào, ghi ở đâu, ai đọc được mà lẽ ra không được.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Điểm ghi chung

```
(log|logger|slog|zap|logrus)\.\w+\(.*(Body|body|payload|params|args|headers?|req\b|request\b)
console\.(log|info|error)\(.*(req\.body|req\.headers|JSON\.stringify)       # TS / JS
(Log::|\$logger->|error_log\().*(\$request->all\(\)|\$_POST|getContent)   # PHP
(logging|logger)\.\w+\(.*(request\.(data|body|POST|headers)|kwargs) # Python
%\+v|%#v|json\.Marshal\(.*(req|input|params)                                # Go: in nguyên struct
audit|activity_log|call_log
```

### Trường nhạy đi vào log

```
(log|logger|console|Log::|logging)\S*\(.*(password|secret|token|otp|cvv|card|authorization|cookie|salary|tax)
```

### Lỗi bên thứ ba, id do client gửi, CRLF

```
(log|logger)\.\w+\(.*(err|error|exception|resp(onse)?\.(Body|text|data))
X-Request-I[Dd]|X-Correlation-I[Dd]|traceparent
Sprintf\(.*%s.*r\.(Header|URL|Form)|`.*\$\{req\.(headers|query|body)
```

### Nhật ký sửa được

```
(UPDATE|DELETE FROM)\s+\w*(audit|activity|history|log)\w*
GRANT\s+ALL|GRANT .*(UPDATE|DELETE).* ON
```

## Examples

### Flag

```go
// Dispatcher chung ghi nguyên tham số; chỉ bỏ khoá tên giống secret → HIGH
func (d *Dispatcher) Call(ctx context.Context, name string, args map[string]any) (any, error) {
    safe := dropKeys(args, "password", "token")
    d.calls.Insert(ctx, name, mustJSON(safe))   // tool nhập lương: {"employee_id":7,"gross":42000000,"tax_code":"..."} nằm nguyên
    return d.tools[name].Run(ctx, args)
}
```

```typescript
// Log body của mọi request, gồm /login và /reset-password → HIGH
app.use((req, _res, next) => { logger.info('req', { path: req.path, body: req.body }); next(); });
```

```php
// Lỗi SMTP mang địa chỉ nhận + phản hồi máy chủ vào log → MEDIUM
catch (TransportException $e) { Log::error('mail failed: ' . $e->getMessage()); }
```

```python
# Log injection: username có "\n2026-01-01 INFO login ok user=admin" → MEDIUM
logging.info("login failed user=" + request.form["username"])

# Khoá nối sự kiện kiểm toán lấy từ header client → MEDIUM
audit.write(correlation_id=request.headers.get("X-Request-Id"), action="export")
```

### Không flag

```go
// Chỉ ghi khung: định danh, kỳ, số phần tử
d.calls.Insert(ctx, name, fmt.Sprintf(`{"period":%q,"rows":%d,"bytes":%d}`, in.Period, len(in.Rows), size))
```

```typescript
logger.info('login failed', { userId: user.id, reason: 'bad_password' });   // không có mật khẩu
logger.info('card saved', { last4: card.number.slice(-4) });
```

## Fix recommendation

1. **Allowlist, không blocklist**: mỗi endpoint / tool khai trường nào được ghi; mặc định chỉ ghi khung (định danh, số phần tử, kích thước). Endpoint lạ chỉ ghi kích thước.
2. Lọc theo tên khoá chỉ là lớp thứ hai.
3. Adapter ngoài (thư, thanh toán) che email / token trong lỗi trước khi trả lên; tầng trên chỉ log mã lỗi + id.
4. Nhật ký chỉ nhận giá trị vô hướng, cắt theo độ dài cột; giá trị lồng ghi "có đổi".
5. Dùng logger có cấu trúc; nếu ghi dòng chữ thì thay `\r`, `\n`, ký tự điều khiển bằng khoảng trắng.
6. Id kiểm toán do server sinh; id của client lưu ở trường riêng, đã giới hạn độ dài và ký tự.
7. User DB của app chỉ `INSERT` / `SELECT` trên bảng nhật ký; kiểm quyền lúc khởi động.
8. Sự kiện kích hoạt từ ngoài có trần theo tài khoản / ngày.
9. Test: gọi endpoint với giá trị đánh dấu rồi tìm giá trị đó trong log và bảng nhật ký, phải không thấy.

Đã lỡ ghi secret vào log: xoay secret, không chỉ xoá dòng log.

## Cross-references

- `01-hardcoded-secret`: secret nằm **trong mã nguồn / config commit**. Rule này: secret / PII **lúc chạy** bị ghi ra log.
- `17-verbose-error-debug-mode`: lộ ra **response** cho client. Rule này: lộ vào **nơi lưu** (file log, bảng nhật ký, hàng đợi). Cùng một `err` vừa trả ra vừa log → hai finding khác dòng.
- `27-excessive-data-exposure`: API trả trường nhạy cho người gọi. Màn xem nhật ký trả trường nhạy cho vai thấp → báo rule này nếu gốc là nhật ký ghi thừa.
- `18-missing-rate-limit`: spam sự kiện để làm đầy nhật ký.
