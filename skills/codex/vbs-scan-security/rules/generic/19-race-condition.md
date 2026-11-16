---
id: RACE-CONDITION
severity_max: HIGH
applies_to: all
cwe: [CWE-362, CWE-367, CWE-833]
---

# Race Condition (TOCTOU): tiền, kho, trần số lượng, khoá, OTP

## Intent

**Time-of-check vs time-of-use (TOCTOU)**: code đọc state → check điều kiện → cập nhật state, **giữa 3 bước có concurrent request khác chen vào**. Kết quả: double-spend, oversell, infinite coupon, withdraw vượt balance.

Pattern điển hình:
```python
balance = get_balance(user)        # đọc 100
if balance >= amount:              # check 100 >= 50 OK
    balance -= amount              # 100 - 50 = 50
    save_balance(user, balance)    # ghi 50
```

Nếu 2 request `amount=50` chạy song song, cả hai đều thấy `balance=100`, cả hai trừ về 50, kết quả: trừ 1 lần dù rút 2 lần → mất 50.

Vibe coder hay sinh code "đọc → if → ghi" không bọc transaction / row lock.

Không chỉ tiền / kho: cùng mẫu này làm vượt trần số lượng, vượt trần thử OTP, thao tác quyền chạy trên trạng thái đã đổi. Khoá sai thứ tự cũng là lỗi (deadlock).

## Khi nào HIGH

- Endpoint financial / wallet / payment / withdrawal có pattern read-modify-write **không** transaction
- Inventory / stock decrement không atomic
- Coupon / discount redeem không check race (1 coupon dùng được nhiều lần)
- Quota / credit usage không atomic
- File rename / move có check `if exists` rồi mới rename (symlink race)
- **Trần số lượng "đếm rồi chèn"**: `SELECT COUNT` → so trần → `INSERT` không khoá, trần gắn với tiền / bảo mật (số ghế trả phí, số thiết bị tin cậy, số lần dùng mã)
- **OTP / mã dùng một lần**: so mã xong mới ghi lượt sai → N request song song thử N mã trong một nhịp, vượt trần khoá
- **Kiểm ở transaction này, ghi ở transaction khác**: đọc vai / trạng thái của đích rồi gọi câu ghi / xoá không điều kiện; giữa hai bước đích đổi vai → thao tác quyền chạy trên trạng thái cũ

## Khi nào MEDIUM (giảm cấp)

- Race trên non-financial counter (likes, views) — sai số chấp nhận được
- Code có lock nhưng lock không bao phủ đúng critical section
- Code dùng optimistic locking (version column) nhưng không retry → user gặp error
- **Thứ tự khoá ngược nhau**: thao tác A khoá X rồi Y, thao tác B khoá Y rồi X → deadlock, user thường làm thao tác của người khác trả 500
- **Tạo lười gặp UNIQUE**: INSERT lỗi trùng → SELECT lại **trong cùng transaction** (REPEATABLE READ không thấy) → 404 hoặc rollback thao tác bao ngoài
- Đọc rồi ghi cấu hình không khoá → lịch sử ghi "giá trị cũ" sai
- Trần đếm rồi chèn không gắn tiền, bảo mật

## Khi nào KHÔNG tạo finding

- Có UNIQUE / CHECK constraint làm chốt cuối và code xử lý lỗi trùng đúng.
- `UPDATE … WHERE <điều kiện>` rồi kiểm số dòng đổi; `INSERT … ON CONFLICT`; tăng đếm nguyên tử.
- Khoá (`FOR UPDATE`, advisory lock) bao trọn đọc-kiểm-ghi. Mutex trong tiến trình chỉ đủ khi app chạy một instance.
- "Có thể race" mà không chỉ ra **hai request nào, chen ở đâu, hệ quả gì** → hardening note.
- Lọt cả khi gọi **tuần tự** (webhook gửi lại) → BUSINESS-STATE-INTEGRITY.

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Grep** tìm hàm liên quan tiền/quota: `balance`, `wallet`, `credit`, `stock`, `inventory`, `quota`, `coupon`, `withdraw`, `transfer`, `redeem`
2. **Read** handler đó:
   - Có `BEGIN TRANSACTION` / `db.transaction()` / `with transaction.atomic():` không?
   - Có `SELECT ... FOR UPDATE` hoặc `WHERE balance >= ?` trong UPDATE không?
   - Hay là pattern `SELECT → check → UPDATE`?
3. **Trace concurrent**:
   - Endpoint async chấp nhận nhiều request đồng thời (mặc định Node, Go)
   - Single-thread (Python WSGI sync) vẫn race ở DB level
4. **Atomic operation check**: `UPDATE wallet SET balance = balance - 50 WHERE user_id = ? AND balance >= 50` — đây là an toàn (atomic). Còn select-modify-update không atomic.
5. **Trần số lượng**: `COUNT` đứng trước `INSERT` trong cùng hàm — có khoá dòng cha hoặc constraint không?
6. **OTP**: lượt thử ghi **trước** hay sau khi so mã? Ghi + đếm có chung transaction khoá dòng không?
7. **Ranh giới transaction**: hàm kiểm và hàm ghi chung `tx` không? Câu ghi có lặp lại điều kiện (`AND status = …`) không?
8. **Thứ tự khoá**: các hàm khoá ≥ 2 bảng / dòng — so thứ tự giữa các hàm.
9. **Tạo lười**: `ensure…` / `getOrCreate…` trong transaction — nhánh lỗi trùng đọc lại bằng gì?

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Pattern read-modify-write

```
# Tìm get balance / get stock
get_?(balance|stock|inventory|quota|credit|wallet)
findOne.*\.balance|findOne.*\.stock
SELECT\s+balance\s+FROM
.balance\s*-=|.stock\s*-=|.credit\s*-=
```

### Tìm thiếu lock

```
# Tìm các nơi update tiền nhưng KHÔNG có transaction
\.update\([^)]*balance|\.save\(\)
UPDATE\s+\w+\s+SET\s+balance\s*=\s*\?  # update absolute value (warn)
```

### Đếm rồi chèn, OTP, tạo lười, thứ tự khoá

```
COUNT\(|\.count\(\)|->count\(\)|\.Count\(          # sau đó có INSERT / create?
(attempts?|tries|failed_?count)\s*(\+\+|\+=)|verify_?(otp|code|totp)
(ensure|getOrCreate|get_or_create|firstOrCreate|findOrCreate)\w*
1062|23505|Duplicate entry|IntegrityError|ER_DUP_ENTRY
FOR UPDATE|lockForUpdate|select_for_update   # so thứ tự khoá giữa các hàm
```

### Lock / transaction patterns (an toàn)

```
SELECT\s+.*\s+FOR\s+UPDATE
\.transaction\(|begin\s*\(\)
with\s+transaction\.atomic
db\.session\.begin_nested
db\.transaction\s*\{
SERIALIZABLE
```

## Examples

### HIGH — flag

```javascript
// Express — wallet withdraw không lock
app.post('/api/withdraw', async (req, res) => {
  const user = await User.findById(req.user.id);
  if (user.balance >= req.body.amount) {
    user.balance -= req.body.amount;
    await user.save();                       // race: 2 request đồng thời cùng pass
    await Payout.create({ userId: user.id, amount: req.body.amount });
  }
  res.json({ ok: true });
});
```

```python
# Django — stock decrement không atomic
def buy(request, product_id):
    product = Product.objects.get(id=product_id)
    if product.stock > 0:
        product.stock -= 1
        product.save()                       # oversell
        Order.objects.create(user=request.user, product=product)
```

```python
# Coupon redeem race
def redeem(code):
    c = Coupon.objects.get(code=code)
    if not c.used:
        c.used = True
        c.save()                              # 2 request cùng dùng 1 coupon
        user.credit += c.amount
        user.save()
```

```javascript
// File rename TOCTOU
if (!fs.existsSync(target)) {                 // check
    fs.writeFileSync(target, data);           // use → giữa hai bước attacker tạo symlink
}
```

```go
// Trần số ghế: đếm rồi chèn, không khoá → 2 request cùng thấy 9/10, thành 11 ghế
n, _ := repo.CountMembers(ctx, orgID)
if n >= plan.Seats { return ErrSeatLimit }
return repo.AddMember(ctx, orgID, userID)
```

```typescript
// OTP: so xong mới ghi lượt sai → 50 request song song thử 50 mã, cùng thấy attempts = 0
const rec = await Otp.findOne({ where: { userId } });
if (rec.attempts >= 5) throw new Locked();
if (rec.code !== input) { await rec.increment('attempts'); throw new BadCode(); }
```

```php
// Kiểm ở ngoài, xoá không điều kiện ở transaction khác
if ($target->role === 'owner') abort(403);
DB::table('members')->where('user_id', $target->id)->delete();   // giữa hai bước $target được nâng owner
```

```go
// MEDIUM — thứ tự khoá ngược: Transfer khoá accounts rồi ledgers, Close khoá ledgers rồi accounts → deadlock
```

```python
# MEDIUM — tạo lười: gặp UNIQUE rồi đọc lại trong cùng transaction (REPEATABLE READ) không thấy
try: Setting.objects.create(org=org)
except IntegrityError: return Setting.objects.get(org=org)   # DoesNotExist → 404
```

### NOT critical — không flag

```javascript
// Atomic SQL update với điều kiện
const result = await db.query(
  'UPDATE wallet SET balance = balance - $1 WHERE user_id = $2 AND balance >= $1',
  [amount, userId]
);
if (result.rowCount === 0) throw new Error('Insufficient funds');
```

```python
# Django transaction + select_for_update
from django.db import transaction
def withdraw(request):
    with transaction.atomic():
        user = User.objects.select_for_update().get(id=request.user.id)
        if user.balance < amount:
            raise ValidationError("Insufficient")
        user.balance -= amount
        user.save()
        Payout.objects.create(user=user, amount=amount)
```

```python
# Stock decrement atomic
from django.db.models import F
updated = Product.objects.filter(id=product_id, stock__gte=1).update(stock=F('stock') - 1)
if not updated:
    raise OutOfStock()
```

## Fix recommendation

1. **Atomic UPDATE với điều kiện:**
   ```sql
   UPDATE wallet
     SET balance = balance - :amount
     WHERE user_id = :uid AND balance >= :amount;
   -- check affected rows; nếu 0 → insufficient
   ```
2. **SELECT ... FOR UPDATE** trong transaction (pessimistic lock):
   ```python
   with transaction.atomic():
       w = Wallet.objects.select_for_update().get(user_id=uid)
       if w.balance < amount: raise
       w.balance -= amount
       w.save()
   ```
3. **Optimistic locking** với version column:
   ```sql
   UPDATE wallet SET balance = ?, version = version + 1
     WHERE id = ? AND version = ?;
   ```
   Retry khi version mismatch.
4. **Idempotency key** cho payment / withdraw: client gửi UUID, server reject duplicate.
5. **Database constraint**: `CHECK (balance >= 0)` — DB tự reject nếu race xảy ra.
   - Trần số lượng: khoá dòng cha (`SELECT … FOR UPDATE` trên tổ chức / gói) rồi mới đếm và chèn.
   - OTP: **giữ lượt trước khi so mã** — ghi lần thử + đếm + bật khoá trong một transaction khoá dòng theo tài khoản; mã đúng thì gỡ.
   - Kiểm và ghi chung một transaction, khoá các dòng quyết định, câu ghi lặp lại điều kiện trạng thái, số dòng đổi = 0 thì báo lỗi.
   - Mọi hàm khoá nhiều bảng theo **một thứ tự cố định**.
   - Tạo lười: coi lỗi trùng là "đã có"; cần đọc lại thì chạy lại cả thao tác ở transaction mới. Test N luồng cùng gọi.
6. **Queue serialization**: route tất cả op của 1 user qua 1 worker (Redis stream, partition by user_id).
7. **Audit log** mọi balance change — phát hiện sớm khi inconsistency xảy ra.

## Cross-references

- Cross-check với `12-broken-access-control`: race + thiếu auth = nhân đôi vấn đề
- Cross-check với `18-missing-rate-limit`: race xảy ra dễ khi attacker spam request
- Cross-check với `02-sql-injection` (nếu có): cả hai liên quan DB layer
- `28-business-state-integrity`: lỗi lọt cả khi gọi tuần tự (webhook xử lý lại, chuyển trạng thái không kiểm nguồn) → rule 28. Chỉ lọt khi song song → rule này
- `06-brute-force`: OTP không có trần lượt thử → rule 06. Có trần nhưng song song vượt được → rule này
- `30-auth-session-lifecycle`: thu hồi phiên chen với thao tác đang chạy
