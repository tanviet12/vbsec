# Data Flow Classification — Trust Levels L1-L4

Phân loại nguồn dữ liệu để quyết định 1 pattern code có thực sự là vulnerability hay không. **Đây là phương pháp luận cốt lõi của vbsec** — phân biệt giữa pattern-match thuần và phân tích bảo mật thực thụ.

## Bốn mức tin cậy

| Level | Tên | Tin cậy | Mô tả |
|---|---|---|---|
| **L1** | User-controlled input | **KHÔNG TIN** | Bất cứ thứ gì user (kể cả attacker) có thể gửi vào |
| **L2** | Database / persistent storage | Bán tin | Giá trị từ DB — có thể gốc gác từ L1 user input đã lưu trước đó |
| **L3** | Internal code / config | Tin | Hardcoded strings, config keys (đã verify), computed value từ trust source |
| **L4** | System / runtime | Tin | Env vars, file path nội bộ, framework constants |

## L1 — Untrusted sources (LUÔN coi là attacker-controlled)

### Web frameworks
- **Express/Node**: `req.body`, `req.query`, `req.params`, `req.headers`, `req.cookies`, `req.files`
- **Next.js**: `req.body` trong API routes, `searchParams`/`params` trong app router, `context.params/query` trong `getServerSideProps`
- **Fastify**: `request.body`, `request.query`, `request.params`, `request.headers`
- **NestJS**: parameters decorated với `@Body()`, `@Query()`, `@Param()`, `@Headers()`
- **Express middleware**: `req.user` SAU khi auth (đã verify) là L3; TRƯỚC auth vẫn là L1
- **Flask**: `request.form`, `request.args`, `request.json`, `request.cookies`, `request.headers`, `request.files`
- **Django**: `request.GET`, `request.POST`, `request.FILES`, `request.COOKIES`, `request.META['HTTP_*']`
- **FastAPI**: function parameters bound qua Pydantic from request body (BEFORE validation chạy là L1; AFTER Pydantic validate là L2-ish — vẫn cần escape trong sink)
- **Go net/http**: `r.URL.Query()`, `r.FormValue`, `r.PostFormValue`, `r.Header`, `r.Body`, `mux.Vars(r)`
- **Go Gin**: `c.Query`, `c.PostForm`, `c.Param`, `c.GetHeader`, `c.ShouldBind*`
- **Go Echo**: `c.QueryParam`, `c.FormValue`, `c.Param`, `c.Request().Header.Get`
- **Go Fiber**: `c.Query`, `c.Body`, `c.Params`, `c.Get` (header), `c.FormFile`
- **PHP**: `$_GET`, `$_POST`, `$_REQUEST`, `$_COOKIE`, `$_FILES`, `$_SERVER` (headers như `HTTP_*`, `REQUEST_URI`), `php://input`, `file_get_contents("php://input")`
- **Laravel**: `$request->input()`, `$request->all()`, `$request->query()`, `$request->json()`, `Input::get()`, route bindings BEFORE validation
- **Ruby/Rails**: `params[...]`, `request.body`, `request.headers`, `cookies[...]`
- **Spring (Java)**: `@RequestParam`, `@RequestBody`, `@PathVariable`, `@RequestHeader`, `@CookieValue`
- **CLI**: `os.Args` (Go), `sys.argv` (Python), `process.argv` (Node), `$argv` (PHP CLI), `ARGV` (Ruby)

### Bot/messaging
- Telegram bot: `update.Message.Text`, `message.text`, `message.caption`, callback data
- Discord/Slack bot: message content, slash command args, button payloads
- Webhook payloads: bất kỳ field nào từ external service nhưng attacker có thể spoof (nếu signature verify failed/missing → vẫn L1)

### File system / streams
- Stdin (đọc từ pipe trong CLI tool)
- Uploaded file content (kể cả filename — `req.file.originalname`)
- File path do user nhập (qua `?path=...` query param)

### Network
- DNS responses (có thể bị poison)
- External HTTP API responses **mà user control URL** (qua SSRF)

### App di động (Flutter / React Native / native)
- **Payload push**: `RemoteMessage.data`, `message.notification`, `NotificationResponse.payload` (`flutter_local_notifications`). Server của mình gửi, nhưng ai có FCM server key / quyền gửi (hoặc backend bị lạm dụng) cũng gửi được → L1 khi dùng làm đường dẫn, URL, tên file
- **Deep link / intent**: `GoRouterState.uri`, `state.pathParameters`, `state.uri.queryParameters`, `app_links` / `uni_links` `uriLinkStream`, `getInitialLink`, `PlatformDispatcher.defaultRouteName`. App hay trang web bất kỳ trên máy mở được → L1 thật
- **Clipboard**: `Clipboard.getData`, `Pasteboard`
- **File từ app khác**: `receive_sharing_intent`, `file_picker`, `image_picker`, file zip / backup người dùng chọn (cả nội dung lẫn tên file)
- **WebView**: `JavaScriptChannel.onMessageReceived`, `onNavigationRequest` URL — trang trong WebView gửi lên
- Response API của chính backend: L2 (tin như DB). Thành L1 khi dùng làm đường dẫn file, URL mở WebView, HTML render

## L2 — Database / persistent storage

Dữ liệu từ DB, Redis, S3... LUÔN có thể chứa L1 đã lưu trước đó. Treat như semi-trust:

- **Safe to use trong query** (parameterized) — DB engine xử lý đúng kể cả khi value chứa SQL syntax
- **NOT safe to interpolate vào HTML** — Stored XSS xảy ra khi L2 chứa `<script>` được render thẳng
- **NOT safe trong `exec`/shell** — nếu L2 chứa shell metacharacters → command injection
- **NOT safe khi build URL** — nếu L2 chứa redirect target → open redirect / SSRF

**Quy tắc:** L2 dùng trong cùng class sink với nguồn gốc của nó là OK (DB→DB query). L2 sang class khác (DB→HTML/shell/URL) thì treat như L1.

## L3 — Internal code

Tin tưởng được:

- Hardcoded strings, regex patterns, SQL fragments không có placeholder
- Computed values từ L3/L4 sources (vd: `sessionKey := fmt.Sprintf("user:%s", userID)` — nếu `userID` là L3 hoặc đã verified từ JWT là L3)
- Application config sau khi đã load và validate
- Internal IDs do app tự sinh (UUID v4, sequence từ DB)

**Cẩn thận:** Nếu L3 ghép với L1 và toàn bộ ghép vào sink, bộ ghép trở thành L1.

## L4 — System / runtime

Tin tưởng tuyệt đối:

- Environment variables (loaded at startup)
- File paths cố định trong code
- Framework/library constants
- Build-time variables

**Lưu ý:** Env var **không** được log/print ra response → đó là cách lộ secret (HARDCODED-SECRET rule check).

---

## Phân tích data flow (workflow cho LLM agent)

Khi gặp 1 pattern code khả nghi (vd: `db.query` có string concatenation), làm theo bước:

```
1. IDENTIFY SINK
   └─ Đây là sink loại gì? SQL? HTML? Shell? URL? File path?

2. IDENTIFY DATA INPUT
   └─ Value nào ghép vào sink? Biến tên gì?

3. TRACE BACKWARD (read full function, có thể đi qua nhiều file)
   └─ Biến đó assign từ đâu?
       ├─ Trực tiếp từ req.X / $_GET / params → L1
       ├─ Từ DB query result → L2
       ├─ Từ hardcoded string → L3
       ├─ Từ os.Getenv → L4
       └─ Từ kết hợp: dùng level cao nhất (KHÔNG TIN > tin)

4. CHECK SANITIZATION ON PATH
   └─ Giữa source và sink có:
       ├─ Parameterization? (db.query(..., [val]) — placeholder)
       ├─ Escape/encode? (HTML escape, URL encode, JSON.stringify)
       ├─ Whitelist validate? (regex match, enum check)
       └─ Type cast? (parseInt — loại bỏ non-numeric)

5. VERDICT
   ├─ L1 → sink không có sanitization → FLAG (CRITICAL/HIGH tùy rule)
   ├─ L1 → sink có sanitization phù hợp → SAFE, không flag
   ├─ L2/L3/L4 → sink → tùy class sink (xem quy tắc L2)
   └─ Không rõ → đọc thêm context, nếu vẫn không rõ thì flag với severity giảm 1 cấp + note "uncertain"
```

## Ví dụ ứng dụng

### Ví dụ 1: SQL Injection — flag CRITICAL

```python
@app.route("/user/<id>")
def get_user(id):              # id từ URL → L1
    sql = f"SELECT * FROM users WHERE id = {id}"  # L1 ghép vào SQL string
    return db.execute(sql)     # sink = SQL, không parameterize
```

**Verdict:** L1 → SQL sink, không sanitization → **CRITICAL** SQL-INJECTION.

### Ví dụ 2: Sprintf safe — không flag

```go
func GetSession(userID string) {  // userID đã từ JWT verified upstream → L3
    key := fmt.Sprintf("session:%s", userID)
    db.Where("key = ?", key).First(&session)  // parameterized
}
```

**Verdict:** L3 + parameterized → SAFE. Không flag dù có `fmt.Sprintf` quanh SQL liên quan.

### Ví dụ 3: Stored XSS — flag HIGH

```jsx
function Comment({ id }) {
  const data = useFetch(`/api/comments/${id}`)  // data từ DB → L2
  return <div dangerouslySetInnerHTML={{__html: data.body}} />  // L2 vào HTML sink
}
```

**Verdict:** L2 dùng cross-class (DB→HTML). Nếu `body` có thể chứa user input (rất khả năng) → **HIGH** stored XSS.

### Ví dụ 4: Command Injection — flag CRITICAL

```javascript
app.get('/ping', (req, res) => {
  const host = req.query.host    // L1
  exec(`ping -c 1 ${host}`, ...)  // L1 vào shell sink
})
```

**Verdict:** L1 → shell sink → **CRITICAL** COMMAND-INJECTION.

### Ví dụ 5: Trust boundary có sanitization — không flag

```python
@app.post("/user")
def create_user(user: UserCreate):  # UserCreate là Pydantic schema
    # user.email đã được Pydantic validate qua EmailStr → format ok
    # nhưng nội dung vẫn từ L1 → khi insert DB, vẫn cần parameterized
    db.execute("INSERT INTO users (email) VALUES (:email)", {"email": user.email})
```

**Verdict:** L1 → SQL sink CÓ parameterization → SAFE.

---

## Quy tắc rút gọn

> **Một dòng code có pattern nguy hiểm + L1 input + không sanitization phù hợp = vulnerability thực sự.**
>
> **Một dòng code có pattern nguy hiểm + chỉ L3/L4 = không phải vulnerability.**

Đây là lý do tại sao vbsec phải READ context, không chỉ grep. Pattern-match thuần sẽ vừa miss thật vừa fp ầm ầm.
