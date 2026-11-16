# Dart / Flutter Specialization

Overlay cho `primary_language: dart` (đuôi `.dart`, marker `pubspec.yaml`). Trọng tâm là **app Flutter** (client iOS / Android): code chạy trên máy người dùng, mọi thứ đóng gói vào app coi như công khai, nguồn L1 là payload push, deep link / intent, clipboard, file từ app khác (xem `references/data-flow-classification.md` mục "App di động").

## Rules

| Rule | File | Focus |
|---|---|---|
| HARDCODED-SECRET | `01-hardcoded-secret.md` | Secret server trong app (`.env` asset của `flutter_dotenv`, `String.fromEnvironment`, FCM server key, service account, `.p8`, keystore); token / mật khẩu lưu `SharedPreferences` / file thay vì `flutter_secure_storage`; key client Firebase **không** phải secret |
| XSS | `03-xss.md` | WebView nạp URL / HTML ngoài kèm `JavaScriptChannel` / token, `url_launcher` mở URL từ push / deep link không whitelist, Flutter Web `innerHtml` |
| SLOPSQUATTING | `05-slopsquatting.md` | Tên package pub.dev bịa / sai chính tả, `git:` không khoá ref, hosted lạ, phân biệt gói `_plus` của Flutter Community |
| PATH-TRAVERSAL | `10-path-traversal.md` | Payload push / deep link ghép vào `context.go('/x/$id')`, ZipSlip với `package:archive`, tên file ngoài ghép vào `File(...)` |
| BROKEN-ACCESS-CONTROL | `12-broken-access-control.md` | Đăng xuất / 401 không xoá state trong RAM, 401 chỉ nhận ở một số verb (đăng xuất kẹt), `allowBackup` / `dataExtractionRules`, deep link / component exported không dùng |
| VERBOSE-ERROR-DEBUG-MODE | `17-verbose-error-debug-mode.md` | `print` / `debugPrint` / Dio `LogInterceptor` không bọc `kDebugMode`, `badCertificateCallback`, `usesCleartextTraffic`, `NSAllowsArbitraryLoads`, `<certificates src="user"/>`, `debuggable`, route debug lọt vào release |
| OUTDATED-DEPENDENCY | `20-outdated-dependency.md` | Đọc `pubspec.lock`, danh sách CVE Pub đã đối chiếu OSV.dev (`archive`, `dio`, `http`, `jose`), Dart SDK cũ |

`--sca` hỗ trợ ecosystem `Pub` (đọc `pubspec.lock`, xem `references/dependency-scan.md`). `--auto-fix` verify bằng `dart analyze --no-fatal-warnings <file>`; bump package Pub luôn chỉ là gợi ý patch.

## Rule generic giữ nguyên

`SQL-INJECTION`, `IDOR`, `BRUTE-FORCE`, `MASS-ASSIGNMENT`, `INSECURE-DESERIALIZATION`, `SSRF`, `CSRF`, `WEAK-PASSWORD-HASHING`, `JWT-NONE-ALGORITHM`, `CORS-MISCONFIG`, `UNRESTRICTED-FILE-UPLOAD`, `MISSING-RATE-LIMIT`, `RACE-CONDITION`, `COMMAND-INJECTION` dùng bản generic. Với app Flutter thuần client, các rule thuần server này **không tạo finding ở code app** (xem `references/language-detection.md` mục "App di động Flutter"); chỉ áp khi repo có backend Dart (`shelf`, `dart_frog`, `serverpod`).

## Hardening note thường gặp (không phải finding)

- File tạm chia sẻ / sao chép ảnh biên lai không tự dọn (`01-hardcoded-secret.md`).
- Thiếu lớp che app switcher cho app hiện số dư (`12-broken-access-control.md`).
- Thiếu certificate pinning (`17-verbose-error-debug-mode.md`).
- Key client Firebase chưa hạn chế theo package / bundle id (`01-hardcoded-secret.md`).

## Nguồn

Rút từ một đợt soát bảo mật app Flutter thật (ví điện tử, nhận thông báo đẩy): token SharedPreferences → Keychain, xoá phiên sạch, 401 ở POST, `allowBackup`, deep link không dùng, `transaction_id` từ push ghép vào route, `.env` asset, file tạm chia sẻ.

## Reasoning

Overlay không thay phân tích L1-L4. Với mỗi pattern: lần giá trị về nguồn (push, deep link, file ngoài, response API), kiểm lọc / gate (`kDebugMode`, whitelist, regex theo định dạng API) ở **mọi** đường vào, chỉ tạo finding khi có đường khai thác.
