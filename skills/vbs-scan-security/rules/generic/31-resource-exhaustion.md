---
id: RESOURCE-EXHAUSTION
severity_max: HIGH
applies_to: all
cwe: [CWE-400, CWE-409, CWE-770, CWE-1333]
owasp: [API4:2023]
---

# Resource Exhaustion (chi phí mỗi request)

## Intent

**Một** request rẻ với người gửi nhưng đắt với server: vài KB bung thành hàng trăm MB, một chuỗi làm regex chạy hàng chục giây, một tham số `limit` kéo cả bảng vào RAM.

Khác MISSING-RATE-LIMIT: rule đó hỏi "gọi **bao nhiêu lần**", rule này hỏi "**mỗi lần** tốn bao nhiêu".

## Khi nào HIGH

Người chưa đăng nhập hoặc user thường gửi được, và **một request** đủ chiếm hàng trăm MB RAM / đĩa hoặc treo CPU nhiều giây:

- **Bom giải nén qua body**: framework / proxy tự bung `Content-Encoding: gzip | br | deflate` khi handler đọc body, trần body chỉ tính byte nén. Hoặc code tự `gzip.NewReader` / `zlib.decompress` rồi đọc hết không trần.
- **Chặn bằng header lọt**: code từ chối `Content-Encoding` nhưng chỉ đọc **một** dòng (dòng đầu), tầng giải nén dùng dòng khác → gửi hai header là qua.
- **Zip / tar do user tải lên** giải nén không trần tổng kích thước thật, số entry (tin `UncompressedSize` trong header là tin attacker).
- **Đọc body / file upload vào RAM không trần**: `io.ReadAll(r.Body)`, `req.on('data')` cộng dồn, `php://input`, `file.read()`, multipart không `limits`; repo không có trần ở tầng nào.
- **ReDoS**: regex có lượng từ lồng (`(a+)+`, `(\w+\s?)+$`) hoặc regex do user gửi, chạy trên input không cắt độ dài, engine quay lui (JS, PHP PCRE, Python `re`).

## Khi nào MEDIUM

- `limit` / `page_size` từ client không có trần; export không phân trang trên bảng tăng theo dữ liệu.
- Bộ lọc khoảng thời gian không giới hạn độ rộng + `LIKE '%…%'` → quét toàn bảng.
- Mỗi request đọc **toàn bộ** bảng vào RAM rồi lọc trong code, bảng lớn dần theo người dùng.
- Trần sai đơn vị: trần từng trường nhưng không trần số phần tử; đếm ký tự trong khi cột giới hạn byte sau mã hoá → người quyền thấp nhồi dữ liệu làm thao tác của người khác lỗi.
- JSON / GraphQL lồng sâu không giới hạn độ sâu, số phần tử.
- Các mục HIGH nhưng cần vai tin cậy, hoặc đã có trần nhưng lỏng.

## Khi nào KHÔNG tạo finding

- Vấn đề là **số lượt gọi** (thiếu rate limit, spam OTP, brute force) → MISSING-RATE-LIMIT / BRUTE-FORCE.
- Có trần thấy được trong repo: `MaxBytesReader`, `LimitReader`, `BodyLimit`, `express.json({ limit })`, multer `limits`, `client_max_body_size`, `MAX_CONTENT_LENGTH`, `post_max_size`. Trần body **không** che bom giải nén nếu bung sau khi đếm.
- Mặc định framework đã có trần (Express `json()` 100kb, Django 2,5MB) và code không nới.
- Go `regexp` (RE2): tuyến tính → không flag ReDoS. Regex trên input đã cắt ngắn cũng không.
- Endpoint chỉ admin, job nội bộ, CLI, script migrate.
- Bảng nhỏ cố định (danh mục, cấu hình) đọc hết.
- Loại file nguy hiểm → UNRESTRICTED-FILE-UPLOAD. Entry zip có `../` → PATH-TRAVERSAL.
- "Query này có thể chậm", "thiếu index", N+1 → không phải lỗ hổng. Chỉ flag khi nêu được **input cụ thể** và **bậc chi phí** (KB → GB, độ dài n → n²).

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Tìm trần toàn cục trước**: cấu hình server, middleware body limit, proxy trong repo. Trần tính trên byte nén hay đã bung?
2. **Body nén**: framework có tự giải nén không (Fiber, middleware `decompress`)? Sau khi bung có `LimitReader` không? Chặn bằng header thì đọc mọi dòng hay một dòng?
3. **Archive**: vòng lặp entry có cộng dồn byte **thực đọc** và dừng ở trần không; có trần số entry không.
4. **Đọc vào RAM**: mỗi `ReadAll` / `.read()` trên input từ ngoài — trần ở đâu?
5. **Phân trang**: `limit` từ query tới SQL có qua `min(limit, MAX)` không?
6. **Regex**: engine gì; có lượng từ lồng không; input dài tối đa bao nhiêu.
7. **Quét toàn bảng**: `findAll()` / `SELECT *` không WHERE rồi lọc trong code.
8. Finding ghi: ai gửi được, input cỡ nào, server tốn gì.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Giải nén

```
Content-Encoding|gzip\.NewReader|brotli|zlib\.(decompress|gunzip)|gzdecode|gzuncompress|gzinflate
zip\.(NewReader|OpenReader)|tar\.NewReader|ZipArchive|extractTo|zipfile|tarfile|extractall|adm-zip|yauzl
UncompressedSize|file_size              # tin kích thước khai trong header?
(Peek|Get)\(.Content-Encoding.\)        # đọc một dòng header
```

### Đọc không trần

```
io\.ReadAll|ioutil\.ReadAll|c\.Body\(\)|BodyParser            # Go
req\.on\(.data.|express\.(json|raw)\(\)|multer\(\s*\)          # TS / JS
php://input|getContent\(\)                                   # PHP
request\.(body|data|get_data)|\.read\(\)                      # Python
```

### Trần (an toàn)

```
MaxBytesReader|LimitReader|BodyLimit|limits?\s*:|client_max_body_size|MAX_CONTENT_LENGTH|post_max_size
```

### Phân trang, quét bảng

```
(limit|page_?size|per_?page|take)\s*[:=]+.*(req\.|r\.URL|c\.Query|\$request|\$_GET|request\.(args|GET))
findAll\(\s*\)|::all\(\)|\.objects\.all\(\)|SELECT \* FROM \w+\s*["'`;)]
```

### Regex

```
\([^)]*[+*][^)]*\)[+*]                        # lượng từ lồng
new RegExp\(.*(req|input|query)|preg_match\(\s*\$|re\.(compile|match|search)\(.*request
```

## Examples

### Flag

```go
// Fiber: BodyLimit 1MB tính byte nén; c.Body() tự bung gzip / br không trần → HIGH (webhook công khai)
app := fiber.New(fiber.Config{BodyLimit: 1 << 20})
app.Post("/hooks/inbound", func(c *fiber.Ctx) error {
    var ev Event
    return json.Unmarshal(c.Body(), &ev)
})

// "Đã chặn" nhưng chỉ xem dòng header đầu; gửi 2 dòng Content-Encoding là lọt → HIGH
if string(c.Request().Header.Peek("Content-Encoding")) != "" { return c.SendStatus(415) }
```

```python
# Giải nén zip tải lên: không trần tổng kích thước, số file → HIGH
with zipfile.ZipFile(request.files["archive"]) as z:
    z.extractall(workdir)
```

```typescript
// limit từ client không trần → MEDIUM
const rows = await repo.find({ where: { orgId }, take: Number(req.query.limit) });

// ReDoS: lượng từ lồng trên input không cắt → HIGH
if (/^([a-zA-Z0-9]+\s?)+$/.test(req.body.name)) { /* ... */ }
```

```php
// Mỗi lượt xem đọc mọi bản chụp rồi lọc trong PHP → MEDIUM
$latest = Snapshot::all()->where('project_id', $id)->sortByDesc('created_at')->first();
```

### Không flag

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
zr, _ := gzip.NewReader(r.Body)
data, err := io.ReadAll(io.LimitReader(zr, 8<<20+1))   // trần sau giải nén
if len(data) > 8<<20 { http.Error(w, "too large", 413); return }
```

```typescript
const take = Math.min(Number(req.query.limit) || 20, 100);
app.use(express.json({ limit: '200kb' }));
```

```go
var slug = regexp.MustCompile(`^([a-z0-9]+-?)+$`)   // Go RE2: tuyến tính
```

## Fix recommendation

1. **Trần sau giải nén**: bọc reader đã bung bằng `LimitReader` / đếm byte thực đọc; vượt thì 413. Client của mình không gửi body nén → từ chối mọi request có `Content-Encoding` khác `identity`, duyệt **mọi** dòng header, đặt trước mọi route.
2. **Archive**: trần tổng byte thực giải nén, số entry, kích thước mỗi entry, độ sâu; giải nén theo luồng, dừng ngay khi vượt.
3. **Body / upload**: trần ở cả proxy lẫn app; stream ra đĩa / object storage thay vì RAM.
4. **Phân trang**: `min(limit, MAX)` ở server, mặc định nhỏ; export chạy job nền, stream theo lô.
5. **Khoảng thời gian**: giới hạn độ rộng, bắt buộc cận dưới.
6. **Regex**: bỏ lượng từ lồng, cắt độ dài input trước khi so; engine tuyến tính (RE2) cho mẫu từ user; đặt timeout.
7. **Trần đúng đơn vị**: đo byte sau mã hoá theo cột đích; trần cả số phần tử lẫn tổng kích thước, kiểm ở cửa ghi.
8. Test gửi byte HTTP thô (hai dòng header, chữ thường) và payload nén tỷ lệ cao.

## Cross-references

- `18-missing-rate-limit`: tần suất gọi. Rule này: chi phí **mỗi** request. Endpoint vừa đắt vừa không giới hạn tần suất → finding ở rule này, ghi thêm thiếu rate limit trong `issue_summary`.
- `16-unrestricted-file-upload`: loại / nội dung file (webshell). Rule này: kích thước và bom giải nén.
- `10-path-traversal`: zip slip (tên entry thoát thư mục).
- `22-vulnerable-dependency`: thư viện giải nén / parser có CVE DoS — báo ở rule 22; cấu hình dùng sai của mình báo ở đây.
- `28-business-state-integrity`: người quyền thấp nhồi dữ liệu làm bước chốt của người khác thất bại.
