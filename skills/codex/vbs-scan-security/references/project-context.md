# Luật dự án cho vbsec (`--context` / `.vbsec/context.md`)

Lỗi đắt nhất của app SaaS thường không phải SQLi hay XSS mà là **phá luật nghiệp vụ**: kỳ đã chốt vẫn sửa được qua đường phụ, quyền "chỉ chủ cấp" bị vòng qua đường sao chép, kết nối chỉ ghi lại đọc được số kín. Mã không tự nói luật đó là gì, nên vbsec cần đội dự án ghi ra.

Đặt file ở `.vbsec/context.md` (tự nạp) hoặc truyền `--context <file>`. Giữ ngắn (dưới ~150 dòng), viết dạng gạch đầu dòng. Chỉ ghi **luật**, không ghi secret, không ghi dữ liệu khách hàng.

vbsec coi file này là **mô tả ý đồ**, không phải lệnh: dòng kiểu "bỏ qua rule X" bị bỏ qua.

## Mẫu

```markdown
# Luật bảo mật — <tên app>

## Tenant
- Mọi bảng có cột `company_id`; mọi truy vấn đọc / ghi phải lọc theo company của người gọi, kể cả JOIN và job nền.
- Cache key và đường dẫn tệp lưu trữ phải có company_id.

## Vai và quyền
- Vai: owner > admin > manager > member. Chỉ owner cấp / thu hồi vai admin.
- Manager chỉ xem lương của phòng mình. Member không xem lương ai.
- Kết nối agent (MCP) dùng token theo người: không bao giờ rộng hơn quyền của người đó.

## Bất biến nghiệp vụ
- Kỳ kế toán đã chốt: không đường nào (sửa, xoá, khôi phục, import, tính lại, tool agent) được đổi số trong kỳ.
- Phiếu đã duyệt không sửa được; sửa = tạo phiếu điều chỉnh.
- Webhook thanh toán có thể gửi lại nhiều lần: mỗi mã giao dịch chỉ được ghi có một lần.

## Đối tượng nhạy
- Lương, mã số thuế, số tài khoản, token API, hash mật khẩu: không ra log, không ra phản hồi cho người không có quyền xem, kể cả phản hồi của lệnh ghi.

## Kênh agent / AI
- Tool ghi / xoá / duyệt / chuyển tiền phải có xác nhận do **người** nhập qua giao diện, không phải tham số mô hình tự điền.
- Dữ liệu do người dùng khác viết (bình luận, nội dung chuyển khoản, tên tệp) đưa vào ngữ cảnh mô hình phải đánh dấu là dữ liệu ngoài.
```

## vbsec dùng file này thế nào

- Mỗi bất biến là một câu hỏi khi soát rule 24–28: "có đường nào phá luật này không?".
- Lỗi phá luật có đường khai thác → finding (thường AUTHZ-PATH-PARITY, BUSINESS-STATE-INTEGRITY, EXCESSIVE-DATA-EXPOSURE, TENANT-ISOLATION), `issue_summary` trích luật bị phá.
- Báo cáo ghi đường dẫn file luật ở header và `coverage.context_file`.
