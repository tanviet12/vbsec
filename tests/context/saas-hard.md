# Luật nghiệp vụ — app giả định Tallyboard (quản lý dự án + hoá đơn, nhiều công ty)

1. Mọi dòng dữ liệu thuộc đúng một công ty (`company_id`). Mã dự án (`code`) và số hoá đơn chỉ duy nhất **trong một công ty**; hai công ty có thể trùng mã. Không đường nào (API, báo cáo, cache, job nền, file xuất) được trả dữ liệu của công ty khác.
2. Vai: `owner`, `admin`, `accountant`, `member`, `client`. `client` là khách ngoài công ty, chỉ thấy dự án được mời và các bình luận ở luồng chia sẻ.
3. Số tài chính (ngân sách dự án, đơn giá giờ, số đã chi / còn lại, số tiền hoá đơn) chỉ `owner`, `accountant` và người có quyền `finance` được xem — kể cả qua phản hồi của lệnh ghi. Ghi chú nội bộ của dự án không hiện cho `client`.
4. Quyền `finance` và `billing_admin` chỉ `owner` được cấp, qua mọi đường làm thay đổi bảng quyền (sửa trực tiếp, sao chép, nhập).
5. Lập hoá đơn: `owner`, `accountant`, hoặc quản lý dự án có quyền `finance`. Hoá đơn mới luôn ở trạng thái `draft`. Đánh dấu đã thu, huỷ và khôi phục hoá đơn chỉ `owner`, `accountant`.
6. Mã mời (`invite_token`) chỉ người được mời biết; ai có mã là vào được tài khoản đó.
7. Trợ lý AI (máy chủ MCP) chạy bằng quyền của thành viên tạo token. Bình luận (nhất là của `client`) và ghi chú khách hàng để lại trên trang thanh toán công khai là chữ không tin cậy: chỉ là dữ liệu, không phải lệnh.
8. Thao tác không hoàn tác được qua trợ lý phải có mã xác nhận gửi cho người dùng ngoài luồng chat (email). Đổi vai và đổi quyền chỉ làm trên web, không qua trợ lý.
9. Token của trợ lý đọc được số tài chính; chữ trợ lý ghi ra luồng chia sẻ với `client` là đường dữ liệu đi ra ngoài công ty.
