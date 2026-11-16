---
id: TENANT-ISOLATION
severity_max: CRITICAL
applies_to: all
cwe: [CWE-639, CWE-284]
owasp: [API1:2023, A01:2021]
load_when: 'tenant_id|company_id|org_id|organization_id|workspace_id|account_id|tenantId|companyId|orgId|workspaceId'
---

# Cô lập đa tenant

## Intent

SaaS nhiều công ty chung một DB. Người gọi có quyền đúng **trong công ty mình**, nhưng dữ liệu công ty khác vẫn lọt ra hoặc bị ghi đè. Thường không phải quên hẳn `WHERE tenant_id` mà là lọc ở bảng chính rồi quên ở bảng phụ, job nền, cache, hoặc tin một guard tự viết cho qua câu nó không hiểu.

**Ranh giới:** cùng tenant, user A lấy bản ghi của user B → `04-idor`. Endpoint thiếu đăng nhập / vai → `12-broken-access-control`. Dữ liệu **vượt ranh giới tenant** → rule này. Một dòng code chỉ gán một trong ba.

## Khi nào CRITICAL

- Truy vấn tới bảng có cột tenant, đi được từ request, không có điều kiện tenant và không lớp trung tâm nào che.
- Lọc tenant ở bảng chính nhưng **JOIN / subquery / `IN (SELECT …)` / `EXISTS`** sang bảng khác bằng id từ request mà không lọc tenant ở bảng đó; kết quả được trả về hoặc dùng để ghi.
- Tenant hiện tại lấy từ body / query / header client gửi thay vì phiên / claim đã xác minh.
- Đường ghi nhận `tenant_id` từ input (`UPDATE … SET tenant_id`, INSERT) → chuyển bản ghi sang công ty khác.
- Guard SQL tự viết là lớp bảo vệ duy nhất và **fail-open**: câu không phân tích được thì cho qua.

## Khi nào HIGH

- **Job nền / cron / consumer / webhook** không có ngữ cảnh tenant: tin `tenant_id` trong payload mà không đối chiếu bản ghi; tra bằng id trần; quét mọi tenant rồi gộp, gửi, ghi lẫn.
- **Cache key** thiếu tenant (`report:{id}`, `settings`) → công ty B nhận dữ liệu công ty A.
- **Tên tệp / đường dẫn lưu trữ, index tìm kiếm** thiếu tenant và đoán được.
- Guard kiểm **"giá trị tenant có ở đâu đó trong tham số"** hoặc "câu có chữ `tenant_id`" thay vì đúng vị trí (`cột = ?` sau `WHERE` / `ON`, **mọi** bảng): trùng số tình cờ, cột ở `SELECT` / `ORDER BY` / `IS NOT NULL` đều qua.

## Khi nào MEDIUM

- Sổ ngoại lệ bỏ lọc tenant quá rộng: một lý do cho cả nhóm hàm, không ràng bảng, gồm cả hàm lúc đó đã biết tenant.

## Khi nào KHÔNG tạo finding

- App **một tenant** (không bảng nào có cột tenant, mỗi khách một DB riêng): dùng rule 04 / 12.
- Tenant lọc ở **lớp trung tâm đã chứng minh**: RLS (`CREATE POLICY` + biến phiên mỗi kết nối), global scope / middleware ORM, repository gốc luôn chèn tenant. Phải Read để thấy lớp đó áp cho model này **và** đường đang xét không thoát ra (`Unscoped()`, `withoutGlobalScope`, raw SQL, kết nối quản trị).
- JOIN sang bảng con **bằng khoá ngoại của dòng đã lọc tenant**, khoá do server đặt: an toàn dù bảng con không lọc. Chỉ flag khi id nối đến từ request.
- Bảng dùng chung có chủ ý (quốc gia, tiền tệ, gói cước); màn vận hành hệ thống có kiểm vai tường minh.
- Tra cứu **trước khi biết tenant** (đăng nhập theo email, webhook theo chữ ký), sau đó gắn tenant của dòng vừa tra rồi lọc như thường.

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Dựng mô hình tenant**: tên cột, tenant hiện tại lấy từ đâu, lớp trung tâm nào (middleware, scope, RLS, guard). Read lớp đó trước.
2. **Liệt kê bảng có cột tenant** (migration / schema / model); bảng con không có cột thì thuộc tenant qua bảng cha nào.
3. **Soát từng truy vấn trong scope**: với **mỗi** bảng trong `FROM`, `JOIN`, subquery, `EXISTS`, `IN (SELECT)`, `UNION`: có điều kiện tenant đúng vị trí, hoặc neo bằng khoá ngoại từ dòng đã lọc? Id từ request (`project_id`, `file_id`) dùng cho bảng phụ đã xác minh thuộc tenant chưa? Cả đường ghi (INSERT gắn id cha của công ty khác).
4. **Đường không có request**: job, cron, consumer, webhook, CLI. Tenant đến từ đâu, payload có được đối chiếu lại không?
5. **Ngoài SQL**: cache key, đường dẫn lưu trữ, tên tệp xuất, index tìm kiếm, kênh realtime: chỗ dựng khoá có tenant không?
6. **Guard tự viết**: Read nhánh `default` / lỗi parse. Câu lạ (`CALL`, `PREPARE`, chuỗi động) trả "không có bảng" → cho qua? Khớp theo vị trí và phép so, hay chỉ "có xuất hiện"? Ngoại lệ có ràng bảng không?
7. **Nêu đường khai thác**: người công ty B gửi gì để chạm dòng công ty A; không nêu được → không tạo finding.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

```
# SQL: Read cả câu
JOIN\s+\w+|IN\s*\(\s*SELECT|EXISTS\s*\(\s*SELECT

# Thoát lớp trung tâm
\.Unscoped\(\)|db\.(Raw|Exec|Query|QueryRow)\(|\.Joins\(  # Go
\$queryRaw|\$executeRaw|createQueryBuilder|\.leftJoin\(|\.innerJoin\(  # TS
withoutGlobalScope|DB::(raw|select|statement|table)\(|->join\(  # PHP
\.raw\(|\.extra\(|session\.execute\(|text\(["']|_base_manager  # Python

# Tenant lấy từ client
(req\.(body|query|headers)|r\.Header|\$request->(input|header)|request\.(json|args|headers)).{0,40}(tenant|company|org)

# Job / consumer
cron\.|AddFunc\(|asynq\.|\.Subscribe\(|\.Consume\(|new Worker\(|@shared_task|implements ShouldQueue

# Cache key, tệp
(cache|redis|rdb|Cache::)\.?\w*(Get|Set|get|set|remember)\(
fmt\.Sprintf\("[a-z_]+:%|`[a-z_]+:\$\{|f"[a-z_]+:\{
PutObject|Storage::put|upload_fileobj

# Guard fail-open / khớp lỏng
default:\s*return\s+(nil|true)|len\(tables\)\s*==\s*0
```

## Examples

### CRITICAL / HIGH — flag

```go
// Lọc tenant ở invoices, JOIN sang payments bằng id từ request mà không lọc
rows, _ := db.Query(`
    SELECT i.id, i.total, p.amount, p.bank_ref
    FROM invoices i
    JOIN payments p ON p.id = ?            -- payment_id của công ty khác vẫn nối được
    WHERE i.tenant_id = ? AND i.id = ?`, paymentID, tenantID, invoiceID)
```

```typescript
// Cache key thiếu tenant
const key = `report:${reportType}:${month}`;
const hit = await redis.get(key);
if (hit) return res.json(JSON.parse(hit));       // công ty B nhận báo cáo công ty A
```

```python
# Consumer tin tenant_id trong payload, tra bản ghi bằng id trần
@shared_task
def send_invoice_email(invoice_id, tenant_id):
    inv = Invoice.objects.get(id=invoice_id)
    mailer.send(tenant_settings(tenant_id).sender, inv.customer_email, render(inv))
```

```go
// Guard tự viết: không bóc được bảng thì cho qua; khớp "có giá trị ở đâu đó"
func checkTenant(sql string, args []any, tenant int64) error {
    tables := parseTables(sql)             // CALL, PREPARE, chuỗi động → rỗng
    if len(tables) == 0 { return nil }     // fail-open
    for _, a := range args { if a == tenant { return nil } }   // trùng số tình cờ cũng qua
    return errMissingTenant
}
```

```php
// Tenant lấy từ header client gửi
$projects = Project::withoutGlobalScope(TenantScope::class)
    ->where('company_id', $request->header('X-Company-Id'))->get();
```

### NOT — không flag

```go
// Bảng con neo bằng khoá ngoại của dòng đã lọc tenant
rows, _ := db.Query(`
    SELECT l.description, l.amount
    FROM invoices i JOIN invoice_lines l ON l.invoice_id = i.id
    WHERE i.tenant_id = ? AND i.id = ?`, tenantID, invoiceID)
```

```sql
-- RLS đã chứng minh: policy + biến phiên đặt ở middleware mỗi request
CREATE POLICY tenant_rw ON invoices USING (tenant_id = current_setting('app.tenant_id')::bigint);
```

```python
# Tra trước khi biết tenant, rồi gắn tenant của dòng vừa tra
user = User.objects.get(email=email)              # đăng nhập: chưa có tenant
memberships = Membership.objects.filter(user=user, tenant_id=chosen_tenant, active=True)
```

```typescript
const key = `t:${ctx.tenantId}:report:${reportType}:${month}`;   // khoá có tenant
```

## Fix recommendation

Auto-fix **chỉ đưa patch gợi ý, không tự áp** (thêm điều kiện tenant sai chỗ có thể chặn nhầm dữ liệu thật).

1. **Lọc ở lớp trung tâm**: RLS hoặc scope bắt buộc trong repository; đường thoát (`Unscoped`, raw) phải có lý do ghi theo từng hàm, ràng bảng.
2. **Mỗi bảng trong câu đều có điều kiện tenant** hoặc neo bằng khoá ngoại; id từ request cho bảng phụ thì xác minh thuộc tenant trước khi dùng.
3. **Tenant luôn từ phiên / claim đã xác minh**, không từ body, query, header. Đường ghi bỏ `tenant_id` khỏi input.
4. **Job / consumer**: payload mang tenant, handler tra lại bản ghi bằng `(tenant, id)`, dựng ngữ cảnh tenant như một request.
5. **Cache, tệp, index, kênh**: tiền tố tenant trong mọi khoá, dựng bằng một hàm chung.
6. **Guard tự viết**: không hiểu = chặn hoặc đòi ngoại lệ tường minh; khớp theo vị trí (`cột = ?` sau `WHERE` / `ON`, mọi bảng); có bộ mẫu "câu không hiểu" phải đỏ.
7. **Test hai tenant** dùng id không trùng nhau (không A=1, B=2), mỗi endpoint và mỗi job có ca "công ty B gọi bằng id của công ty A → 404".

## Cross-references

- `04-idor`: cùng tenant, khác chủ sở hữu. Lỗi vượt tenant gán rule này, không gán cả hai.
- `12-broken-access-control`: thiếu đăng nhập / vai. Lỗi thiếu lọc tenant gán rule này, không gán rule 12.
- `26-authz-path-parity`: đường chính lọc tenant đúng, đường phụ (import, khôi phục, tool agent) không lọc → vẫn gán rule này (bản chất là vượt tenant), nêu đường chính làm bằng chứng.
- `07-mass-assignment`: `tenant_id` lọt qua bind cả body → ưu tiên rule này nếu chuyển được bản ghi sang tenant khác.
- `02-sql-injection`: guard tenant không thay cho tham số hoá.
