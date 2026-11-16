# Luật nghiệp vụ — ứng dụng sổ quỹ nội bộ (bộ test ledger-hard)

- Kỳ kế toán đã chốt không được đổi bằng bất kỳ đường nào: tạo, sửa, xoá, khôi phục, nhập tệp, tính lại. Số dư chốt là số cuối.
- Kỳ chỉ được chốt khi không còn đề nghị chi nào ở trạng thái chờ duyệt hoặc đã duyệt chưa chi, kể cả của quỹ đang lưu trữ.
- Một bút toán chỉ được tính một lần vào tổng thu / chi của kỳ, dù gắn bao nhiêu nhãn.
- Mỗi sự kiện từ ngân hàng / cổng chi (webhook) chỉ được ghi nhận một lần; bên gửi có thể gửi lại nhiều lần.
- Tổng đề nghị chi đang mở của một phòng trong kỳ không vượt hạn mức tháng của phòng, kể cả khi gửi song song.
- Người duyệt đề nghị chi là trưởng phòng **hiện tại** của phòng; trưởng phòng cũ mất quyền duyệt ngay khi bị thay. Không ai tự duyệt đề nghị của mình.
- Đăng nhập hai bước: mật khẩu → mã 6 số. Sai mã quá 5 lần trong 15 phút thì khoá; phiếu giữa hai bước chỉ dùng một lần.
- Cho nhân viên nghỉ là thu hồi ngay mọi cách truy cập của người đó: phiên web và token API cá nhân.
- Dữ liệu nhạy: lương, mã số thuế, số tài khoản ngân hàng (lưu mã hoá), token, phiếu đăng nhập, link mời. Không được xuất hiện trong log hay nhật ký hoạt động.
- Vai: admin, accountant (kế toán), manager (trưởng phòng), staff. Chỉ admin sửa lương, cho nghỉ, đổi trưởng phòng.
