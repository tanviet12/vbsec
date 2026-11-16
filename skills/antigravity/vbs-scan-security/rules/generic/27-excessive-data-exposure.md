---
id: EXCESSIVE-DATA-EXPOSURE
severity_max: HIGH
applies_to: all
cwe: [CWE-200, CWE-213]
owasp: [API3:2023, A01:2021, LLM02:2025]
---

# Lộ dữ liệu quá mức qua phản hồi

## Intent

Người gọi **được phép gọi** endpoint, nhưng phản hồi mang nhiều hơn thứ họ được xem:

1. Trả nguyên model / dòng DB: kèm hash mật khẩu, secret 2FA, token, trường nghiệp vụ nhạy (lương, số tiền, mã số thuế).
2. **Phản hồi của lệnh ghi thành cửa đọc**: quyền chỉ ghi gửi giá trị rác, phản hồi trả "để tiện" số kế hoạch, số cũ, nguyên bản ghi đã lưu.
3. Cùng dữ liệu, màn này che theo quyền, màn kia (danh sách, tìm kiếm, tool agent, thông báo) không che.
4. Xuất tệp / bản chụp / cache mang giá trị thật ra ngoài phạm vi có luật che.

Khác IDOR: người gọi có quyền với **bản ghi**, không có quyền với **trường**.

## Khi nào HIGH

- Phản hồi (kể cả lồng trong quan hệ nạp kèm) chứa **bí mật xác thực**: hash mật khẩu, secret 2FA, mã khôi phục, API key, refresh token.
- Phản hồi chứa trường nghiệp vụ nhạy cho người **không có quyền xem trường đó**, trong khi code có luật che ở nơi khác (hàm mask, quyền `view_*`): bằng chứng trường này vốn phải che.
- Lệnh ghi (POST / PUT / PATCH / tool ghi / import) trả số mà luật đọc của người gọi chặn: giá trị cũ, số kế hoạch, số dư, bảng đối chiếu.
- Xuất tệp / báo cáo / bản chụp chứa trường bị che, trong khi quyền xuất hoặc quyền đọc bản chụp rộng hơn quyền xem trường.

## Khi nào MEDIUM

- Lệnh ghi chỉ trả **mã so sánh** (`match` / `mismatch`) cho người chỉ ghi, không trần lượt → dò nhị phân ra số.
- Nhật ký thay đổi (cũ → mới) hiện trường bị che cho người sửa không được xem trường đó.
- Bản chụp lưu giá trị, lúc đọc chỉ lọc theo quyền **hiện tại**, bỏ qua luật che phụ thuộc thời điểm chụp.
- Danh sách trả dữ liệu cá nhân của người khác (email, số điện thoại) vượt nhu cầu màn hình; thông báo / webhook mang số kín tới người không được xem.

## Khi nào LOW

- Trang / API đã đăng nhập trả dữ liệu nhạy mà thiếu `Cache-Control: no-store`. Gộp **một finding** cho cả app.

## Khi nào KHÔNG tạo finding

- Trường trả cho **chính chủ** (hồ sơ, lương của tôi) hoặc cho người có quyền xem trường đó.
- Phản hồi qua DTO / serializer **liệt kê trường tường minh**, không có trường nhạy. "Trả cả struct" không phải lỗi nếu struct không có trường nhạy hoặc trường nhạy bị loại lúc serialize (`json:"-"`, `$hidden`, `exclude`): phải Read định nghĩa.
- App **không có quyền xem theo trường** (mọi thành viên xem mọi số theo thiết kế) và trường không phải bí mật xác thực. Không tự đặt luật che mà code và context dự án không có.
- Lệnh ghi trả đúng giá trị người gọi vừa gửi + id + trạng thái.
- Dữ liệu nhạy vào **log** → `29-sensitive-data-in-logs`. Stack trace → `17-verbose-error-debug-mode`.
- Thiếu `no-store` ở trang không có dữ liệu nhạy.

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Lập danh sách trường nhạy.** Bí mật xác thực: luôn nhạy. Trường nghiệp vụ: tìm nơi code **đã che** (hàm `mask*`, `canView*`, quyền `view_salary`, nhánh gán `null` theo vai) hoặc context dự án: mỗi nơi che cho biết một trường có luật xem.
2. **Với mỗi trường nhạy, liệt kê mọi cửa ra**: chi tiết, danh sách, tìm kiếm, xuất tệp, báo cáo, bản chụp / cache, thông báo, webhook, kết quả tool agent, nhật ký thay đổi, và **phản hồi của mọi lệnh ghi**. Cửa không gọi hàm che = ứng viên.
3. **Soát riêng lệnh ghi.** Có vai / token chỉ ghi không? Phản hồi gồm gì ngoài id + trạng thái? Có qua cùng hàm che của đường đọc không? Rà **mọi** lệnh ghi của khu.
4. **Soát serialize.** Handler trả thẳng entity / dòng DB (`SELECT *`, `RETURNING *`)? Read struct / model: tag JSON, danh sách ẩn, quan hệ nạp kèm (`user` lồng trong `comment`).
5. **Bản chụp và xuất.** Lưu giá trị hay tham chiếu? Quyền đọc bản chụp / tệp xuất so với quyền xem trường.
6. **Nêu ai thấy gì**: vai X gọi Y nhận trường Z mà luật ở file:dòng chặn. Không chỉ ra được người nhận thiếu quyền → không tạo finding.
7. **Header cache**: xem middleware chung trước khi kết luận thiếu.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

```
# Trả thẳng model / dòng DB
json\.NewEncoder\(\w+\)\.Encode\(|c\.JSON\(\s*\w+,\s*\w+\)|json:"(password|hash|secret|salary|tax)  # Go
res\.(json|send)\(\s*(user|row|rows|result|saved|updated|record)|RETURNING \*|include:\s*\{  # TS
return \$\w+;|->toArray\(\)|response\(\)->json\(\$\w+\)|makeVisible\(|\$hidden  # PHP
model_to_dict\(|fields\s*=\s*["']__all__|jsonify\(\w+\)|\.dict\(\)|\.model_dump\(\)  # Python
SELECT \*

# Tên trường nhạy
password_?hash|totp_?secret|recovery_?code|api_?key|refresh_?token
salary|payroll|tax_?(id|code)|bank_?account|cost_?price|budget|balance

# Nơi đã che (nguồn của luật)
mask\w*\(|redact\w*\(|canView\w*|can_view_\w+|view_(salary|finance)

# Phản hồi lệnh ghi (Read đoạn return của handler ghi)
(previous|old|planned|expected|variance|compare|reconcil)\w*["']?\s*[:=]

# Xuất / bản chụp / header
export|csv\.NewWriter|xlsx|to_csv|snapshot|Cache-Control|no-store
```

## Examples

### HIGH / MEDIUM — flag

```go
// Trả nguyên struct: password_hash, totp_secret, salary cho mọi thành viên
type Employee struct {
    ID           int64  `json:"id"`
    Name         string `json:"name"`
    PasswordHash string `json:"password_hash"`
    TOTPSecret   string `json:"totp_secret"`
    Salary       int64  `json:"salary"`
}
func (h *Handler) ListEmployees(w http.ResponseWriter, r *http.Request) {
    rows, _ := h.repo.List(r.Context(), tenantOf(r))
    json.NewEncoder(w).Encode(rows)
}
```

```typescript
// Lệnh ghi thành cửa đọc: token chỉ ghi nhận về số kế hoạch và số kỳ trước
router.put("/actuals/:lineId", requireScope("actuals:write"), async (req, res) => {
  const saved = await actuals.upsert(req.tenantId, req.params.lineId, req.body.amount);
  const line = await budget.line(req.tenantId, req.params.lineId);      // đường đọc cần budget:read
  res.json({ ...saved, planned: line.planned, previous: line.lastPeriod, variance: saved.amount - line.planned });
});
```

```python
# Màn chi tiết che lương theo quyền, đường xuất thì không
def employee_detail(request, pk):
    e = get_employee(request, pk)
    return JsonResponse(mask_salary(e, request.user))

def export_employees(request):                     # mọi trưởng nhóm đều có quyền xuất
    rows = Employee.objects.filter(tenant=request.tenant).values()
    return csv_response(rows)                      # salary, tax_code ra nguyên
```

```php
// Bình luận nạp kèm user: hash và email đi theo
return Comment::with('user')->where('task_id', $id)->get();   // model User không khai $hidden
```

### NOT — không flag

```go
// DTO liệt kê trường; lương chỉ điền khi người gọi được xem, kèm cờ "đã che"
type EmployeeDTO struct {
    ID           int64  `json:"id"`
    Name         string `json:"name"`
    Salary       *int64 `json:"salary,omitempty"`
    SalaryMasked bool   `json:"salary_masked"`
}
```

```typescript
// Lệnh ghi chỉ trả định danh + trạng thái + chính giá trị vừa gửi
res.json({ id: saved.id, status: "saved", amount: saved.amount });
```

```python
class UserSerializer(serializers.ModelSerializer):
    class Meta:
        model = User
        fields = ["id", "name", "avatar_url"]          # allowlist, không có trường nhạy
```

## Fix recommendation

Auto-fix **chỉ đưa patch gợi ý, không tự áp** (đổi hình dạng phản hồi có thể làm hỏng client; luật che là luật nghiệp vụ).

1. **DTO / serializer allowlist cho mọi phản hồi**; không trả entity. Bí mật xác thực loại ở tầng model (`json:"-"`, `$hidden`).
2. **Một hàm che ở lõi** nhận (bản ghi, người gọi), mọi cửa ra đều gọi: chi tiết, danh sách, xuất, tool agent, thông báo. API trả cờ "trường bị che" để client không nhầm "che" với "trống".
3. **Phản hồi lệnh ghi đi qua cùng luật đọc** của người gọi: không đọc được thì chỉ trả id + trạng thái; cảnh báo không kèm số. Buộc phải trả mã so sánh cho người chỉ ghi thì đặt trần lượt ghi cùng ô.
4. **Ghi một phần** (PATCH, tool "chỉ gửi ô đổi"): trường người gọi không được xem thì bỏ đầu vào, giữ giá trị DB; nhật ký cũ → mới chỉ ghi trường người đó được xem.
5. **Bản chụp / xuất**: lưu tham chiếu rồi đọc lại qua cửa có luật che; buộc lưu giá trị thì lưu kèm kết quả luật che lúc chụp. Quyền xuất không rộng hơn quyền xem trường.
6. **`Cache-Control: no-store`** ở middleware chung cho mọi phản hồi đã đăng nhập.
7. **Test**: vai không có quyền xem → mọi cửa ra không có trường đó; kết nối chỉ ghi → phản hồi không có số nào khác giá trị nó gửi.

## Cross-references

- `04-idor`: không có quyền với **bản ghi**. Rule này: có quyền với bản ghi, không có quyền với **trường**.
- `26-authz-path-parity`: lệch luật giữa các **đường ghi**. Lệch che giữa các **cửa đọc** → rule này.
- `25-tenant-isolation`: dữ liệu của tenant khác → rule 25.
- `29-sensitive-data-in-logs`: cùng trường nhạy nhưng cửa ra là log.
- `17-verbose-error-debug-mode`: lộ qua thông báo lỗi, stack trace.
- `07-mass-assignment`: chiều ngược (nhận thừa trường). Hay đi cặp: cùng một struct vừa bind vừa trả.
- `24-excessive-agency`: kết nối agent đọc số kín rồi ghi chữ tự do ra ngoài → rule 24.
