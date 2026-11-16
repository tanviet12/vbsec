---
id: BROKEN-ACCESS-CONTROL
severity_max: CRITICAL
applies_to: all
---

# Broken Access Control

## Intent

Backend endpoint **không kiểm tra quyền** — ai gọi cũng chạy. Vibe coder thường giấu nút "Admin" ở frontend nghĩ là an toàn, nhưng `POST /api/admin/delete-user` vẫn mở public. Hacker mở DevTools, copy URL, gọi trực tiếp — xóa hết user, leak database, đổi role thành admin.

OWASP xếp **Broken Access Control là #1** vì AI sinh route nhanh, hay quên `requireAuth` / `@login_required` / `before_action`. Lỗi này không pattern-match được dễ — phải đọc từng route handler.

## Khi nào CRITICAL

- Endpoint sensitive (admin, delete, update user khác, financial) **không có** middleware auth/role
- Endpoint nhận `userId` / `tenantId` từ request body/query mà **không verify** user hiện tại có quyền truy cập resource đó (IDOR)
- Frontend route ẩn nhưng backend mở (`/api/admin/*` không có guard)
- Missing tenant isolation: query không có `WHERE tenant_id = current_user.tenant_id`

## Khi nào HIGH (giảm cấp)

- Endpoint chỉ-đọc (read-only) thông tin không quá nhạy cảm
- Có auth nhưng thiếu role check (user thường truy cập được data user khác cùng quyền)
- Internal API có network-level isolation (chỉ accessible từ VPC)

## Khi nào KHÔNG tạo finding — ghi hardening note

"Endpoint này không có auth" tự nó chưa phải lỗ hổng. Chỉ flag thiếu auth/role check khi có **một trong hai bằng chứng**:

1. **Repo có cơ chế auth và endpoint này bị bỏ sót**: có middleware / decorator auth (`RequireUser`, `@login_required`, `[Authorize]`, `auth.Group`, `requireAdmin`...) dùng ở route khác cùng app, còn route này không có. Đặc biệt chú ý middleware đặt **sai chỗ** (`router.use(requireAdmin)` sau khi đã khai báo route, `before_action` có `except`, group auth không bao hết route).
2. **Bản chất endpoint đòi hỏi quyền**: admin, xoá / sửa dữ liệu người khác, ghi tiền, đọc dữ liệu cá nhân của user khác (email, số dư, đơn hàng), đổi role. Kể cả khi repo chưa có auth nào.

Không có bằng chứng nào trong hai loại trên → **không tạo finding**:

- Repo (hoặc phần code trong scope) **không có cơ chế auth nào** và endpoint chỉ đọc dữ liệu công khai (tìm sản phẩm, xem bài viết, health check, preview): đây là API public hoặc app mẫu chưa có đăng nhập. Ghi **1 hardening note chung** cho cả repo ("chưa có cơ chế auth, cần thêm trước khi đưa dữ liệu nhạy cảm vào"), không tạo finding cho từng endpoint.
- Endpoint **đã có finding CRITICAL/HIGH của rule khác** (COMMAND-INJECTION, SSRF, SQL-INJECTION...) và vấn đề chỉ là **thiếu auth**: thiếu auth chỉ làm lỗi đó nặng hơn, không phải lỗi thứ hai. Ghi "endpoint không cần đăng nhập" vào `issue_summary` của finding chính để giải thích severity, KHÔNG tạo thêm BROKEN-ACCESS-CONTROL cùng dòng. **Ngược lại**, nếu lỗi phân quyền **vẫn còn sau khi sửa lỗi kia** — quyền sở hữu xác định bằng trường user tự đổi được (`display_name`, `email` chưa verify), so sánh theo tên thay vì id, tenant lấy từ body — thì đó là finding riêng, kể cả khi cùng dòng với SQL injection.
- Vấn đề thuộc loại "session không hết hạn", "cookie thiếu cờ", "OAuth thiếu state" mà **không chỉ ra được ai chiếm được gì**: hardening note, không gán rule này.

Nếu reasoning của mình có chữ "có thể", "nên có", "tốt hơn nếu" mà không nêu được attacker lấy được dữ liệu / quyền gì → đó là hardening note.

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Liệt kê routes**: dùng Grep tìm tất cả route definition (`app.get`, `app.post`, `@router.`, `Route::`, etc.)
2. **Read** mỗi route handler:
   - Có middleware auth không? (`requireAuth`, `passport.authenticate`, `@login_required`)
   - Có check role/permission không? (`if user.role !== 'admin'`)
   - Resource ID lấy từ đâu? Nếu từ `req.params.id` / `req.body.userId` → có verify `resource.owner_id === current_user.id` không?
3. **Trace data flow**:
   - L1 (untrusted): `req.params`, `req.body`, query string
   - L4 (trusted): session, JWT claim đã verify
   - Nếu authorization decision dựa trên L1 (vd: `if req.body.isAdmin`) → CRITICAL
4. **Edge case**: GraphQL resolvers, RPC handlers, webhook endpoints thường bị bỏ sót

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Node.js / Express

```
# Routes có thể thiếu auth
app\.(get|post|put|delete|patch)\s*\(\s*["']/(admin|api/admin|users/|delete|update)
router\.(get|post|put|delete)\s*\(

# Tìm dùng requireAuth / isAuthenticated
requireAuth|isAuthenticated|passport\.authenticate|ensureLoggedIn

# IDOR: lấy id từ params/body rồi query trực tiếp
findById\s*\(\s*req\.(params|body|query)
User\.findOne\s*\(\s*\{\s*_id:\s*req\.
```

### Python / Django / Flask

```
# Django views thiếu decorator
^def\s+\w+\s*\(\s*request   # function-based view
class\s+\w+View\s*\(\s*View\) # class-based
# Check có @login_required / @permission_required / LoginRequiredMixin không

# Flask routes
@app\.route\(|@blueprint\.route\(
# Check có @login_required / @jwt_required không

# FastAPI
@app\.(get|post|put|delete)\(   # check Depends(get_current_user)
```

### Rails / PHP / Go

```
# Rails: thiếu before_action :authenticate_user!
class\s+\w+Controller\s+<\s+ApplicationController

# Laravel: middleware('auth') trong route
Route::(get|post|put|delete)\s*\(

# Go: handler không check session
http\.HandleFunc|router\.HandleFunc|\.HandleFunc\(
```

## Examples

### CRITICAL — flag

```javascript
// Express — admin endpoint không có auth
app.post('/api/admin/delete-user', async (req, res) => {
  await User.findByIdAndDelete(req.body.userId);  // ai cũng gọi được
  res.json({ ok: true });
});
```

```python
# Flask — IDOR: lấy user_id từ URL không verify ownership
@app.route('/api/orders/<int:order_id>')
@login_required
def get_order(order_id):
    order = Order.query.get(order_id)  # bất kỳ user nào xem order người khác
    return jsonify(order.to_dict())
```

```python
# Django — view không check role
def admin_panel(request):
    if request.method == 'POST':
        User.objects.all().delete()  # ai vào URL cũng xóa
```

### NOT critical — không flag (hoặc downgrade)

```javascript
// Có middleware auth + role check
app.post('/api/admin/delete-user',
  requireAuth,
  requireRole('admin'),
  async (req, res) => {
    await User.findByIdAndDelete(req.body.userId);
    res.json({ ok: true });
  }
);
```

```python
# Verify ownership trước khi trả data
@app.route('/api/orders/<int:order_id>')
@login_required
def get_order(order_id):
    order = Order.query.get_or_404(order_id)
    if order.user_id != current_user.id:
        abort(403)
    return jsonify(order.to_dict())
```

## Fix recommendation

1. **Thêm auth middleware ở route level:**
   ```javascript
   // Express
   app.use('/api/admin', requireAuth, requireRole('admin'));
   ```
2. **Verify ownership với mọi resource access:**
   ```python
   order = Order.query.filter_by(id=order_id, user_id=current_user.id).first_or_404()
   ```
3. **Dùng policy library** (CASL, Pundit, casbin) để tách logic authorization khỏi handler
4. **Tenant isolation** trong ORM: scope tự động `WHERE tenant_id = ?`
5. **Test:** với mỗi sensitive endpoint, viết test "unauthorized user gọi endpoint → 401/403"

## Cross-references

- Cross-check với `14-jwt-none-algorithm`: nếu JWT yếu thì auth check vẫn vô nghĩa
- Cross-check với `02-sql-injection`: IDOR thường đi kèm SQLi qua param
- Cross-check với `18-missing-rate-limit`: enumeration attack qua IDOR
