# Luật nghiệp vụ — cổng thông tin khách hàng (bộ test misc-hard)

- Mọi endpoint công khai (đăng nhập, quên mật khẩu) phải chịu được request độc hại: kích thước body tính trên dữ liệu **sau giải nén**.
- Tệp nén người dùng tải lên chỉ được bung trong giới hạn số tệp và tổng dung lượng.
- Sau đăng nhập / đổi ngôn ngữ / đăng xuất chỉ được chuyển hướng tới đường dẫn nội bộ của chính cổng.
- Dữ liệu nhạy: email, token phiên, link đặt lại mật khẩu. Không được xuất hiện trong log; log chỉ ghi id người dùng và IP.
- Link đặt lại mật khẩu sống 30 phút, dùng một lần.
