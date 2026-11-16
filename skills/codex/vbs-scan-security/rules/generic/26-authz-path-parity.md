---
id: AUTHZ-PATH-PARITY
severity_max: CRITICAL
applies_to: all
cwe: [CWE-863, CWE-285]
owasp: [API5:2023, A01:2021]
---

# Phân quyền lệch giữa các đường ghi

## Intent

Một đối tượng có nhiều đường ghi. Đường chính (màn sửa, API chuẩn) kiểm đúng: chỉ chủ được cấp quyền này, dòng đã chốt không sửa. **Đường phụ** tới cùng bảng thì không kiểm cùng luật: khôi phục bản cũ, sao chép quyền, import, đồng bộ từ khu khác, hàng loạt, tool cho agent, job nền. Route nào cũng có middleware nên rule 12 không thấy; lỗi chỉ lộ khi **đặt hai đường cạnh nhau mà so**.

Finding luôn có hai vị trí: nơi đường chính đặt luật, nơi đường phụ thiếu luật đó.

## Khi nào CRITICAL

- Đường phụ cho vai thấp **tự nâng quyền hoặc cấp quyền đặc biệt**: quyền "chỉ chủ cấp" đi sang người khác qua "sao chép quyền" / nhân bản tài khoản / mẫu vai / import người dùng kèm vai; cấp được bộ quyền rộng hơn quyền của chính người cấp.
- Đường phụ sửa / xoá được dữ liệu tiền hoặc bản ghi đã khoá, đã duyệt mà đường chính chặn.

## Khi nào HIGH

- Đường phụ ghi vào cùng đối tượng mà thiếu luật đường chính có: chủ sở hữu, vai, phạm vi phòng ban, trạng thái, hạn mức.
- **Luật chỉ đặt ở hộp chọn**: frontend hoặc API "danh sách lựa chọn" lọc mục không được chọn, lệnh ghi nhận id bất kỳ.
- Thao tác làm đối tượng được bảo vệ **biến mất hoặc đổi nghĩa** (ẩn, gộp, xoá dòng cha, chuyển nhóm) không qua luật của đường "sửa".
- **Đồng bộ tự động** từ khu A ghi vào bảng của khu B bằng quyền của A, bỏ qua quyền / trạng thái B đòi khi sửa trực tiếp.
- Tool agent / API công khai gọi thẳng repository, bỏ qua tầng dịch vụ nơi web kiểm quyền.

## Khi nào MEDIUM

- Đường phụ kiểm đúng quyền nhưng thiếu ràng buộc phụ đường chính bắt (lý do, lịch sử thay đổi).
- Chiều ngược: đường **danh sách** lỏng hơn hàm kiểm quyền (hiện cả thành viên đã nghỉ, tài khoản đã khoá), từ đó mở thao tác không đáng có.

## Khi nào KHÔNG tạo finding

- **Mọi đường cùng gọi một hàm lõi** và luật nằm trong hàm đó (đã Read).
- Đường phụ **chặt hơn** hoặc chỉ dành cho vai cao hơn, có kiểm vai tường minh.
- Đường không đi từ input người dùng: migration, seed, test.
- Khác biệt **có chủ ý và ghi rõ** (comment, tài liệu, context dự án).
- **Không nêu được luật cụ thể** đường chính đang thực thi (file:dòng): "đường này nên kiểm thêm" là hardening note.
- Endpoint không có xác thực → `12-broken-access-control`. Chỉ một đường và nó thiếu kiểm chủ sở hữu → `04-idor`.

## Cách reasoning (BẮT BUỘC theo thứ tự — không pattern-match)

1. **Chọn đối tượng đáng soát**: bảng quyền / vai / thành viên, tiền, bản ghi có trạng thái khoá / duyệt, cấu hình dùng chung. Ưu tiên nơi code đã có luật tường minh (`if !isOwner`, `status == "locked"`, `canGrant(...)`): có luật mới có thứ để lệch.
2. **Liệt kê MỌI đường ghi tới đối tượng.** Grep tên bảng, hàm repository, model trên toàn repo. Các loại: web handler; API công khai; hàng loạt; import; khôi phục / hoàn tác; sao chép / nhân bản / mẫu; đồng bộ từ module khác; tool agent / MCP; job / consumer; webhook. Ẩn, gộp, xoá cha, chuyển nhóm cũng là ghi.
3. **Lập bảng so**: mỗi đường → luật kiểm ở đâu (route / handler / lõi / không có) → bằng danh tính nào (người gọi thật hay hệ thống). Đường chính là chuẩn.
4. **Luật ở tầng nào**: hàm ghi lõi mọi đường đi qua → an toàn. Handler hoặc frontend → Read từng đường còn lại, không đoán theo tên.
5. **Hộp chọn so với lệnh ghi**: lệnh ghi có xác minh id gửi lên bằng **cùng hàm** lọc của danh sách không?
6. **Đường chép**: có mang quyền đặc biệt không? Người bấm có được cấp loại quyền đó, có cấp vượt quyền của mình không?
7. **Viết finding**: mỗi đường lệch một finding, nêu (a) đường chính + luật, file:dòng; (b) đường phụ thiếu, file:dòng; (c) vai nào làm được gì mà luật cấm. Thiếu (a) → không tạo finding.

Quét theo diff: vẫn tìm các đường còn lại ngoài diff để so (đọc rộng, chỉ báo lỗi trong scope).

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

```
# Động từ của đường phụ
restore|undo|revert|clone|duplicate|replicate|copy_?(from|perm|role)|template
import|bulk|batch|sync|cascade|merge|archive|hide|move|reassign|recalc

func \(\w+ \*?\w+\) (Restore|Copy|Clone|Import|Bulk|Sync|Merge|Archive)\w*\(  # Go
(router|app)\.(post|put|patch|delete)\([^)]*(restore|copy|clone|import|bulk|sync)  # TS
function (restore|duplicate|copy|import|bulk|sync)\w*\(|->replicate\(|->upsert\(  # PHP
def (restore|clone|copy|import_|bulk_|sync_)\w*\(|bulk_create\(|bulk_update\(  # Python
\.registerTool\(|mcp\.NewTool\(|@mcp\.tool  # tool agent

# Mọi nơi ghi vào bảng đang xét (thay <table>, <Model>)
(INSERT INTO|UPDATE|DELETE FROM)\s+<table>|<Model>(\.objects)?\.(create|update|save|delete)

# Luật ở đường chính
isOwner|is_owner|canGrant|can_grant|authorize\(|Gate::|->can\(|status\s*[!=]=+\s*["'](locked|closed|approved)

# Luật chỉ ở frontend / danh sách chọn
:disabled=|v-if=["'].*can|options\.filter\(|/options|/choices
```

## Examples

### CRITICAL / HIGH — flag

```go
// Đường chính: chỉ chủ được cấp quyền xem tài chính
func (s *PermService) Save(ctx context.Context, actor Actor, target int64, perms []string) error {
    if contains(perms, "finance.view") && !actor.IsOwner { return ErrForbidden }
    return s.repo.Replace(ctx, actor.TenantID, target, perms)
}

// Đường phụ: quản lý bấm "sao chép quyền" → chép nguyên bộ, mang finance.view sang người mới
func (s *PermService) CopyFrom(ctx context.Context, actor Actor, from, to int64) error {
    if !actor.IsManager { return ErrForbidden }
    perms, _ := s.repo.List(ctx, actor.TenantID, from)
    return s.repo.Replace(ctx, actor.TenantID, to, perms)
}
```

```typescript
// Đường chính chặn sửa hoá đơn đã khoá
router.patch("/invoices/:id", auth, async (req, res) => {
  const inv = await invoices.get(req.tenantId, req.params.id);
  if (inv.status === "locked") return res.status(409).end();
  /* ... */
});
// Đường phụ: khôi phục phiên bản cũ ghi đè chính hoá đơn đó, không kiểm trạng thái
router.post("/invoices/:id/versions/:v/restore", auth, async (req, res) => {
  const old = await versions.get(req.tenantId, req.params.id, req.params.v);
  await invoices.overwrite(req.tenantId, req.params.id, old.data);
  res.json({ ok: true });
});
```

```python
# Hộp chọn lọc theo phòng của người gọi…
def approver_options(request):
    return JsonResponse(list(visible_approvers(request.user).values("id", "name")), safe=False)

# …lệnh ghi nhận id bất kỳ
def task_create(request):
    Task.objects.create(tenant=request.tenant, approver_id=request.POST["approver_id"], ...)
```

### NOT — không flag

```go
// Luật nằm ở hàm ghi lõi; mọi đường (lưu, chép, import, tool agent) đều đi qua
func (s *PermService) write(ctx context.Context, actor Actor, target int64, perms []string) error {
    for _, p := range perms {
        if special[p] && !actor.IsOwner { return ErrForbidden }
        if !actor.Has(p) { return ErrGrantBeyondOwn }
    }
    return s.repo.Replace(ctx, actor.TenantID, target, perms)
}
func (s *PermService) CopyFrom(ctx context.Context, actor Actor, from, to int64) error {
    perms, _ := s.repo.List(ctx, actor.TenantID, from)
    return s.write(ctx, actor, to, withoutSpecial(perms))   // đường chép mặc định bỏ quyền đặc biệt
}
```

```php
// Đường phụ chặt hơn đường chính: chỉ quản trị hệ thống, có kiểm vai
public function bulkArchive(Request $r) {
    $this->authorize('system-admin');
    Project::whereIn('id', $r->ids)->where('company_id', $r->user()->company_id)->update(['archived' => 1]);
}
```

## Fix recommendation

Auto-fix **chỉ đưa patch gợi ý, không tự áp** (thêm kiểm quyền bằng máy dễ chặn nhầm người dùng thật).

1. **Đưa luật xuống hàm ghi lõi** mà mọi đường đi qua; handler, tool agent, job chỉ gọi hàm đó. Không để repository ghi thẳng lộ ra ngoài tầng dịch vụ.
2. **Luật cấp quyền gồm hai vế**: ai được cấp loại quyền này, và không cấp vượt quyền của người cấp. Đường chép mặc định bỏ quyền đặc biệt và nói rõ trên giao diện.
3. **Danh sách chọn và lệnh ghi dùng chung một hàm**: lệnh ghi xác minh id gửi lên bằng đúng hàm lọc của danh sách.
4. **Liệt kê mọi thao tác làm bản ghi đã khoá biến mất hoặc đổi nghĩa** (ẩn, gộp, xoá cha, khôi phục, tính lại) và cho đi qua cùng hàm kiểm.
5. **Đồng bộ liên khu**: khoá dòng trạng thái của khu đích trước khi ghi; khu đích đang cần kiểm soát thì chỉ gắn cờ "cần xem lại" cho người có quyền ở đó.
6. **Test theo hệ quả, không theo tên API**: mỗi hàm ghi vào bảng quyền có ca "người mẫu có quyền đặc biệt"; mỗi đường ghi vào bản đã khoá có ca "vai thấp gọi → từ chối"; gọi thẳng lệnh ghi với mục đã bị hộp chọn lọc.

## Cross-references

- `12-broken-access-control`: endpoint **không có** kiểm đăng nhập / vai. Rule này: có kiểm, nhưng **khác luật** so với đường chính. Đường phụ thiếu hẳn middleware → rule 12.
- `04-idor`: một đường, thiếu kiểm chủ sở hữu. Nếu đường chính kiểm chủ sở hữu còn đường phụ không → rule này, nêu cả hai vị trí.
- `25-tenant-isolation`: đường phụ thiếu lọc tenant → rule 25.
- `24-excessive-agency`: tool agent kiểm đúng luật nhưng không cần người xác nhận → rule 24. Tool agent bỏ qua luật của web → rule này.
- `07-mass-assignment`: đường import / bulk bind cả trường `role` → rule 07 nếu gốc là bind không lọc trường; rule này nếu gốc là thiếu luật "ai được cấp".
- `28-business-state-integrity`: sửa số đã chốt mà **không đường nào** chặn đúng → rule 28. Có đường chính chặn, đường phụ quên → rule này.
