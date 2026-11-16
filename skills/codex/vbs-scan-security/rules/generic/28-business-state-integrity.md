---
id: BUSINESS-STATE-INTEGRITY
severity_max: CRITICAL
applies_to: all
cwe: [CWE-840, CWE-841, CWE-837]
owasp: [API6:2023, A04:2021]
---

# Business State Integrity

## Intent

Bất biến nghiệp vụ ("kỳ đã khoá không đổi", "mỗi sự kiện tiền tính một lần") chỉ được kiểm ở **đường chính** (nút Sửa), đường phụ bỏ ngỏ. Hệ quả: số đã chốt đổi không dấu vết, tiền cộng hai lần. Không có sink kỹ thuật: phải hiểu bất biến rồi soát mọi đường ghi.

**Có file luật dự án** (cờ `--context`, hoặc `.vbsec/context.md`): đọc **trước tiên**, lấy bất biến trong đó làm chuẩn, trích câu bị vi phạm vào finding. Không có file: suy bất biến từ code (cột `status`, `locked_at`, hàm kiểm ở đường sửa chính) và ghi rõ "bất biến suy ra từ code".

## Khi nào CRITICAL

- Webhook / callback thanh toán xử lý lại cùng một sự kiện → cộng tiền, kích hoạt đơn nhiều lần: không có khoá chống trùng theo mã giao dịch của nhà cung cấp, hoặc bước "đã xử lý chưa" tách rời bước ghi.
- Chuyển trạng thái có hệ quả tiền mà không kiểm trạng thái nguồn: hoàn tiền đơn chưa thanh toán / đã hoàn, chi phiếu đã huỷ, `paid` đặt theo giá trị từ request.
- Số tiền đã chốt (sổ, hoá đơn đã phát hành, bảng lương đã duyệt) sửa được qua đường phụ bởi người **không có quyền mở lại**.

## Khi nào HIGH

- Bản ghi đã chốt đổi nghĩa hoặc biến mất qua đường phụ: ẩn, gộp, xoá bản ghi cha, khôi phục phiên bản cũ, import. Đường chính có kiểm khoá, đường này không.
- Tính lại (recalculate) ghi đè số kỳ đã chốt, không để lịch sử.
- Nhận dữ liệu mới vào kỳ đã khoá: tạo / import bản ghi có ngày thuộc kỳ khoá, ngày do client gửi.
- Cổng chặn chỉ đếm "phần tử đang bật": tắt / ẩn / lưu trữ phần tử → điều kiện chặn biến mất → chốt → bật lại.
- Đếm đôi: một giao dịch vào hai nhóm của cùng một tổng (JOIN bảng 1-n rồi SUM, tính cả dòng cha lẫn dòng con) làm sai số tiền, hạn mức.
- Nhảy bước: gọi thẳng API bước cuối (duyệt, phát hành) khi bước trước chưa xong.

## Khi nào MEDIUM

- Các mục HIGH nhưng dữ liệu không phải tiền / pháp lý; hoặc chỉ người vốn có quyền mở lại mới đi được đường phụ (mất lịch sử, không leo quyền).
- Người quyền thấp làm bước chốt của người khác thất bại (nhồi dòng để bước gộp vượt cột → rollback).
- Webhook không idempotent, hệ quả chỉ là thông báo / bản ghi trùng.

## Khi nào KHÔNG tạo finding

- Code và file luật đều không có khái niệm chốt / khoá / trạng thái cuối → không đoán bất biến.
- Đường phụ gọi chung hàm kiểm (`ensureOpen`, `assertNotLocked`) hoặc DB có trigger / constraint chặn. Đọc hàm trước khi kết luận.
- Sửa qua luồng "mở lại" có kiểm quyền và ghi lịch sử: đúng thiết kế.
- Webhook INSERT mã sự kiện vào cột UNIQUE **trước** khi xử lý, hoặc `UPDATE … WHERE status = 'pending'` rồi kiểm số dòng đổi: đã idempotent dù không có chữ "idempotency".
- Webhook thiếu kiểm chữ ký → BROKEN-ACCESS-CONTROL.
- Số thống kê không dùng cho tiền / hạn mức (lượt xem) đếm lệch.
- Chỉ lọt khi hai request **song song** chen nhau → RACE-CONDITION.
- "Nên có state machine" mà không nêu được chuỗi bước → hardening note.

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Lập danh sách bất biến**: từ file luật; không có thì grep cột trạng thái / khoá, đọc hàm kiểm của đường sửa chính.
2. **Liệt kê mọi đường ghi** vào bảng có bất biến và bảng cha / con: handler, import, job, đồng bộ, khôi phục, gộp, tính lại, tool của agent. So từng đường với hàm kiểm của đường chính. Hỏi theo hệ quả ("sau thao tác này số đã chốt có đổi không?"), không theo tên API.
3. **Cổng chặn**: câu đếm điều kiện có lọc `enabled` / `deleted_at IS NULL` không? Ai tắt được phần tử?
4. **Kỳ khoá**: ngày của bản ghi mới lấy từ đâu; server có so với kỳ khoá không; import có qua cùng chỗ kiểm?
5. **Webhook**: tìm khoá chống trùng; kiểm và ghi có nguyên tử không? Nhà cung cấp tự gửi lại khi timeout, không cần attacker.
6. **Chuyển trạng thái**: mỗi `SET status = X` có `WHERE status IN (nguồn hợp lệ)` hoặc kiểm trong cùng transaction không?
7. **Tổng tiền**: câu SUM có JOIN bảng 1-n làm một dòng ra hai lần không?
8. Finding nêu: bất biến nào, đường nào, vai nào, chuỗi bước, hệ quả.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Bất biến và hàm kiểm

```
locked_at|closed_at|is_locked|is_closed|finalized|posted_at|period_lock
status\s*(==|===|=|!=)\s*['"](closed|locked|paid|approved|posted|issued|final)
(ensure|assert|check)_?(Open|NotLocked|Editable)
```

### Đường phụ (Go `func`, TS `async` / `function`, PHP `function`, Python `def`)

```
(func|function|def|async)\s.*(restore|revert|merge|archive|hide|bulk|import|sync|recalc|recompute|backfill)
```

### Webhook và chuyển trạng thái

```
webhook|callback|ipn|notify_url
idempotency|event_id|ON CONFLICT|INSERT IGNORE|ON DUPLICATE
UPDATE\s+\w+\s+SET\s+status\s*=        # có WHERE status = nguồn không?
\.(status|state)\s*=\s*(req|request|\$request|body|input)
```

### Đếm đôi

```
SUM\(.*JOIN\s+\w*(tag|label|group|categor|item|line)
```

## Examples

### Flag

```go
// Đường chính có kiểm kỳ khoá
func (s *EntryService) Update(ctx context.Context, e Entry) error {
    if err := s.periods.EnsureOpen(ctx, e.PostedOn); err != nil { return err }
    return s.repo.Update(ctx, e)
}
// Đường phụ: khôi phục phiên bản cũ ghi thẳng, không kiểm kỳ → HIGH
func (s *EntryService) RestoreVersion(ctx context.Context, id, ver int64) error {
    old, _ := s.versions.Get(ctx, id, ver)
    return s.repo.Update(ctx, old.Entry)
}
```

```typescript
// Webhook thanh toán: kiểm và ghi tách rời, không khoá chống trùng → CRITICAL
app.post('/hooks/payment', verifySignature, async (req, res) => {
  const order = await Order.findByPk(req.body.order_id);
  await Wallet.increment('balance', { by: req.body.amount, where: { userId: order.userId } });
  await order.update({ status: 'paid' });        // gửi lại sự kiện → cộng lần hai
  res.sendStatus(200);
});
```

```php
// Cổng chặn chỉ đếm mục đang bật → tắt mục chưa rà rồi chốt → HIGH
$pending = Item::where('period_id', $id)->where('enabled', 1)->where('reviewed', 0)->count();
if ($pending === 0) { $period->update(['closed_at' => now()]); }
```

```python
# Đếm đôi: hoá đơn có 2 nhãn ra 2 dòng → tổng công nợ sai → HIGH
total = db.execute("""SELECT SUM(i.amount) FROM invoices i
    JOIN invoice_tags t ON t.invoice_id = i.id WHERE i.customer_id = %s""", [cid])
```

### Không flag

```typescript
// INSERT mã sự kiện vào cột UNIQUE trước, trùng thì dừng
try { await ProcessedEvent.create({ eventId: req.body.id }); }
catch (e) { if (isUniqueViolation(e)) return res.sendStatus(200); throw e; }
```

```go
// Chuyển trạng thái có điều kiện nguồn + kiểm số dòng đổi
res, _ := tx.Exec(`UPDATE refunds SET status='done' WHERE id=? AND status='approved'`, id)
if n, _ := res.RowsAffected(); n == 0 { return ErrInvalidState }
```

## Fix recommendation

Auto-fix **chỉ đưa patch gợi ý, không tự áp** (nhóm logic: sửa máy dễ chặn nhầm thao tác hợp lệ).

1. Một hàm ghi duy nhất cho bảng có bất biến; mọi đường (sửa, ẩn, gộp, khôi phục, import, tính lại, tool agent) đi qua hàm đó. Hàm tự so cũ – mới và ghi lịch sử khi giá trị đã chốt đổi, bất kể ai gọi.
2. Kỳ khoá kiểm ở tầng ghi theo ngày của bản ghi (server tính), không ở form.
3. Cổng chặn đếm cả phần tử đang tắt nhưng đã từng đủ điều kiện, hoặc cấm tắt khi còn việc chặn.
4. Webhook: bảng sự kiện đã xử lý có UNIQUE theo mã của nhà cung cấp, INSERT trước rồi xử lý trong cùng transaction.
5. Chuyển trạng thái bằng `UPDATE … WHERE status = <nguồn>` và kiểm số dòng đổi.
6. Tổng tiền: SUM trên bảng gốc, lọc nhóm bằng `EXISTS` thay vì JOIN 1-n.
7. Test theo hệ quả: chốt kỳ, chạy từng đường phụ, số đã chốt phải giữ nguyên.

## Cross-references

- `26-authz-path-parity`: lệch **quyền** giữa các đường ghi (ai được làm). Rule này: lệch **bất biến trạng thái** (làm lúc nào, trên bản ghi nào). Đường phụ vừa thiếu quyền vừa phá khoá kỳ → báo rule này nếu hệ quả là số đã chốt đổi.
- `19-race-condition`: chỉ lọt khi song song → rule 19. Lọt cả khi gọi tuần tự (webhook gửi lại) → rule này.
- `12-broken-access-control`: webhook không kiểm chữ ký, endpoint không kiểm quyền.
- `07-mass-assignment`: `status` / `paid` gán từ body qua bind hàng loạt → rule 07; gán tường minh nhưng không kiểm trạng thái nguồn → rule này.
