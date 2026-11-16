---
id: AUTH-SESSION-LIFECYCLE
severity_max: HIGH
applies_to: all
cwe: [CWE-613, CWE-294, CWE-308, CWE-672]
owasp: [A07:2021, API2:2023]
---

# Auth / Session Lifecycle

## Intent

Quyền được cấp đúng lúc đầu nhưng **không chết đúng lúc**: token giữa các bước đăng nhập dùng lại được, phiên sống tiếp sau khi tài khoản bị khoá, quyền đặc biệt quay về khi kết nối lại, người duyệt cũ vẫn duyệt.

**Điều kiện bắt buộc của mọi finding**: nêu được (1) **ai cầm** token / quyền cũ (người đã nghỉ, kẻ lấy được token trung gian, app đã gỡ, người phụ trách cũ), (2) **lấy hoặc làm được gì**. Thiếu một ý → hardening note. "Token không hết hạn" tự nó không phải finding.

## Khi nào HIGH

- **Token trung gian** (sau mật khẩu, trước 2FA / chọn tổ chức) dùng nhiều lần, hoặc cùng dạng với access token nên API thường chấp nhận → bỏ qua 2FA.
- **Không thu hồi**: đăng xuất, cho nghỉ, khoá tài khoản mà access token vẫn chạy nhiều giờ vì server chỉ tin claim (vai lấy từ JWT, không đọc lại DB).
- **Reset / đổi mật khẩu không huỷ phiên, refresh token** đang sống → kẻ đã chiếm phiên vẫn giữ quyền.
- **Quyền đặc biệt gắn theo ứng dụng** (`client_id`) thay vì theo lần cấp phép: gỡ rồi kết nối lại, token cấp lại đều thừa hưởng quyền đã duyệt. Tương tự: mở lại tài khoản đã nghỉ mang nguyên quyền cũ.
- **Người duyệt chụp lúc tạo việc**: quyền duyệt kiểm bằng cột lưu sẵn (`approver_id`); đổi người phụ trách không tính lại → người cũ vẫn duyệt được.
- **Thao tác nhạy không xác thực lại** trong app có 2FA: đổi email, đổi mật khẩu không hỏi mật khẩu cũ, tắt 2FA, tạo API key, rút tiền → phiên bị chiếm thành mất hẳn tài khoản.
- **Scope của token** chỉ kiểm ở một số nhánh → token chỉ-ghi đọc được dữ liệu.

## Khi nào MEDIUM

- Không thu hồi nhưng access token hạn ngắn (≤ ~15 phút) và refresh token đã bị huỷ.
- Thu hồi không phủ thứ người đó **đã phát hành**: lời mời, link chia sẻ, API key còn sống sau khi người tạo bị khoá.
- Đổi người phụ trách mà việc chờ duyệt không chuyển: người mới không thấy, việc treo.
- Refresh token không xoay, dùng lại token cũ không bị phát hiện.
- Token trung gian dùng một lần nhưng không gắn người / thiết bị, hạn dài.

## Khi nào KHÔNG tạo finding

- Không thoả điều kiện bắt buộc: "session dài", "JWT không có `exp`", "cookie thiếu cờ" mà không chỉ ra ai cầm, lấy được gì.
- Middleware **mỗi request** đọc lại phiên / user từ DB, hoặc có denylist, `token_version` → thu hồi đã có. Đọc middleware trước khi kết luận.
- App không có 2FA, không có khái niệm xác thực lại → không flag "thiếu step-up" (trừ đổi mật khẩu không hỏi mật khẩu cũ).
- Người duyệt tính **lúc duyệt** từ cơ cấu hiện tại (JOIN bảng phòng ban) → an toàn dù có cột chụp để hiển thị.
- Thuật toán, chữ ký, `alg: none` → JWT-NONE-ALGORITHM.
- Thử mật khẩu / OTP nhiều lần → BRUTE-FORCE.
- Endpoint không kiểm quyền gì → BROKEN-ACCESS-CONTROL. Rule này chỉ khi quyền **từng đúng** rồi không được thu hồi / tính lại.

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Luồng đăng nhập**: mỗi bước phát token gì, bước sau kiểm gì. Token trung gian có `aud` / loại riêng, `jti` dùng một lần không? Thử: gửi nó vào API thường, gửi lại lần hai.
2. **Middleware xác thực**: sau khi verify chữ ký có tra DB (phiên còn sống, tài khoản hoạt động) không? Vai lấy từ claim hay DB?
3. **Sự kiện phải thu hồi**: đăng xuất, reset mật khẩu, khoá, cho nghỉ, gỡ tích hợp. Mỗi sự kiện có xoá phiên, refresh token, API key, kết nối app không?
4. **Quyền được duyệt riêng**: khoá của bảng quyền là gì (`client_id` + user hay id của lần kết nối)? Kết nối lại có tạo dòng mới chưa duyệt không?
5. **Cột chụp**: grep `approver_id`, `assignee_id`. Liệt kê mọi thao tác đổi **nguồn** của nó (đổi trưởng nhóm, cho nghỉ, đổi phòng cha). Thao tác nào không tính lại?
6. **Thao tác nhạy**: handler có đòi mật khẩu hiện tại / mã 2FA / mốc xác thực gần đây không?
7. Finding theo mẫu: "Sau khi <sự kiện>, <ai> vẫn cầm <gì>, dùng để <làm gì>".

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

### Token trung gian, phát token

```
(mfa|2fa|pre_?auth|challenge|pending|temp)_?[Tt]oken|login_?ticket
jwt\.(sign|encode|NewWithClaims)|JWT::encode|create_access_token|jti|token_type
```

### Middleware và thu hồi

```
(jwt\.verify|ParseWithClaims|JWT::decode|jwt\.decode)      # sau đó có tra DB không?
claims\.(Role|role|IsAdmin|scope)|payload\.role|\$decoded->role|claims\[.role.\]
logout|revoke|invalidate|token_version|denylist|sessions?\.(delete|destroy)
(deactivate|suspend|offboard|remove_?member|lock_?user|(reset|change)_?[Pp]assword)   # có huỷ phiên / token?
```

### Quyền theo ứng dụng, cột chụp, thao tác nhạy

```
client_id|integration_id|grant|elevated|reactivate|reinstate
approver_id|assignee_id|reviewer_id|manager_id
(disable|remove)_?(2fa|mfa|totp)|change_?email|create_?api_?key|reauth|confirm_?password
```

## Examples

### Flag

```go
// Bước 1 phát token cùng khoá ký, cùng claim với access token; không jti, không aud → HIGH
func (s *Auth) PasswordStep(u User) string {
    return sign(jwt.MapClaims{"sub": u.ID, "exp": time.Now().Add(10 * time.Minute).Unix()}, s.key)
}
// Middleware API chỉ verify chữ ký + exp → token bước 1 gọi được mọi API, bỏ qua 2FA
```

```typescript
// Cho nghỉ: chỉ đổi cờ. Middleware tin claim, access token hạn 7 ngày → HIGH
async function offboard(userId: string) { await db.user.update({ where: { id: userId }, data: { active: false } }); }
export const auth = (req, _res, next) => { req.user = jwt.verify(bearer(req), KEY); next(); };
```

```php
// Reset mật khẩu không huỷ phiên / token khác → HIGH
$user->password = Hash::make($request->password);
$user->save();
```

```python
# Quyền nhập lương duyệt theo (client_id, user): gỡ app rồi nối lại vẫn có quyền → HIGH
def can_import_payroll(conn):
    return ElevatedGrant.objects.filter(client_id=conn.client_id, user_id=conn.user_id).exists()

# Người duyệt chụp lúc gửi; đổi trưởng nhóm không cập nhật → người cũ vẫn duyệt → HIGH
def approve(request, req_id):
    r = LeaveRequest.objects.get(id=req_id)
    if r.approver_id != request.user.id: raise PermissionDenied
```

### Không flag

```go
// Mỗi request dựng lại người gọi từ DB
sess, err := s.sessions.GetActive(ctx, claims.SessionID)   // phiên bị xoá khi đăng xuất / cho nghỉ
member, err := s.members.GetActive(ctx, sess.UserID, sess.OrgID)   // vai lấy từ DB
```

```php
$user->save();
DB::table('sessions')->where('user_id', $user->id)->delete();
$user->tokens()->delete();
```

"JWT hạn 30 ngày" trong app không có chức năng khoá tài khoản, không nêu được ai cầm token → hardening note, không finding.

## Fix recommendation

Auto-fix **chỉ đưa patch gợi ý, không tự áp** (nhóm quyền: sửa máy dễ khoá nhầm người dùng thật).

1. Token trung gian: `aud` / loại riêng, khoá ký dẫn xuất riêng, `jti` dùng một lần (ghi DB), hạn vài phút; API thường từ chối loại này.
2. Mỗi request dựng lại người gọi từ DB (phiên còn sống, tài khoản hoạt động, còn là thành viên); vai từ DB, không từ claim.
3. Một hàm "thu hồi mọi thứ của người này" (phiên, refresh token, API key, kết nối app, lời mời đã gửi), gọi ở mọi sự kiện: đăng xuất tất cả, reset mật khẩu, khoá, cho nghỉ.
4. Quyền được duyệt riêng khoá theo lần cấp phép (dòng kết nối); kết nối mới phải duyệt lại. Mở lại tài khoản = vào mới về quyền.
5. Cột chụp người xử lý: mọi thao tác đổi nguồn gọi chung hàm "tính lại + chuyển + báo" trong cùng transaction; hoặc kiểm quyền duyệt theo cơ cấu lúc duyệt.
6. Thao tác nhạy đòi mật khẩu hiện tại hoặc 2FA trong N phút gần nhất.
7. Scope kiểm trước mọi nhánh; mỗi lệnh khai mức đọc / ghi.

## Cross-references

- `14-jwt-none-algorithm`: token **giả mạo được** (chữ ký, thuật toán). Rule này: token **thật** nhưng sống quá vòng đời.
- `06-brute-force`: đoán mật khẩu / OTP. Rule này: dùng lại thứ đã được cấp.
- `12-broken-access-control`: endpoint thiếu kiểm quyền từ đầu. `26-authz-path-parity`: đường này kiểm, đường kia không. Rule này: kiểm đúng nhưng trên dữ liệu quyền đã cũ (claim, cột chụp, grant theo app).
- `19-race-condition`: thu hồi và dùng token chen nhau trong cùng thời điểm.
