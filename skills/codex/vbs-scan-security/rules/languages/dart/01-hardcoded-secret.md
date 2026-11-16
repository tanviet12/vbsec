---
id: HARDCODED-SECRET
severity_max: CRITICAL
applies_to: dart
overrides: generic/01-hardcoded-secret.md
---

# Hardcoded Secret / Lưu bí mật sai chỗ — Dart / Flutter

> Override cho `rules/generic/01-hardcoded-secret.md` khi `primary_language: dart`. Vẫn áp mọi pattern key của generic (prefix `sk_live_`, `AKIA`, `ghp_`, `xoxb-`, `sk-ant-`, `-----BEGIN ... PRIVATE KEY-----`, gán `password`/`secret`/`token` = chuỗi dài).

## Intent (Flutter-specific)

App di động là **client**: mọi thứ đóng gói vào app đều coi như công khai. Ai tải app từ store cũng giải nén được `.ipa` / `.apk`, đọc asset, chạy `strings` lên `libapp.so` / `App.framework`. Vì vậy với Flutter có hai loại lỗi:

1. **Bí mật server nằm trong app**: secret trong code Dart, trong `--dart-define`, trong `.env` khai làm asset (`flutter_dotenv`). Không commit cũng vẫn lộ, vì nó nằm trong bản build phát cho người dùng.
2. **Bí mật của người dùng lưu sai chỗ**: access token, refresh token, mật khẩu, PIN lưu `SharedPreferences` / file / Hive / sqflite không mã hoá, thay vì Keychain (iOS) / Keystore (Android) qua `flutter_secure_storage`. `SharedPreferences` iOS là file plist trong container app (đi theo bản sao lưu máy tính / iCloud), Android là XML trong `shared_prefs/` (đi theo bản sao lưu nếu `allowBackup`, máy root đọc được).

## Khi nào CRITICAL

- Secret **phía server** trong code Dart, asset hoặc `String.fromEnvironment(..., defaultValue: '<secret>')`: secret key cổng thanh toán, API key AI có tính tiền, khoá ký HMAC request, mật khẩu DB, JWT signing secret, **FCM server key** (legacy `AAAA....:APA91b...`), JSON **service account** (`"type": "service_account"`, `"private_key"`), file **`.p8`** (APNs / App Store Connect), keystore ký app (`*.jks`, `*.keystore`) kèm `key.properties` có mật khẩu.
- `.env` (hoặc file tương tự) khai trong `flutter: assets:` của `pubspec.yaml` mà chứa secret server: đóng gói nguyên văn vào app.

## Khi nào HIGH

- Access token / refresh token / mật khẩu / PIN / OTP seed lưu bằng `SharedPreferences`, `GetStorage`, Hive box không `encryptionCipher`, sqflite, `File(...).writeAsString` thay vì `flutter_secure_storage` (hoặc Keychain/Keystore qua platform channel).
- `flutter_secure_storage` với iOS `synchronizable: true` cho token (đồng bộ iCloud Keychain sang máy khác).

## Khi nào KHÔNG tạo finding

- **Cấu hình client Firebase**: `apiKey` (`AIza...`) trong `firebase_options.dart`, `google-services.json`, `GoogleService-Info.plist`, cùng `appId`, `projectId`, `messagingSenderId`. Đây là định danh công khai, bảo vệ bằng Security Rules / App Check / hạn chế key theo package + SHA-1, bundle id. Ghi **hardening note** "kiểm hạn chế key trên Google Cloud Console", không gán rule. Tương tự: Google Maps key client, Sentry DSN, Mapbox `pk.` public token.
- `.env` asset chỉ chứa giá trị không bí mật (`BASE_URL`, cờ tính năng): hardening note ("asset `.env` là cửa để secret lọt vào bundle sau này, nên bỏ").
- Token chỉ giữ trong RAM (biến của provider / model), không ghi đĩa.
- FCM device token, user id, cài đặt giao diện lưu `SharedPreferences`: không phải bí mật.

## Cách reasoning

1. Grep secret như generic, cộng thêm `String.fromEnvironment`, `dotenv.env[`, `dotenv.get(`, file `.env*` và `flutter: assets:` trong `pubspec.yaml`.
2. Với mỗi giá trị: là secret **server** (cho phép gọi API với quyền của công ty / tốn tiền) hay định danh **client** công khai? Chỉ loại đầu là finding.
3. Với lưu trữ: grep `SharedPreferences`, `setString(`, `GetStorage`, `Hive.openBox`, `writeAsString`, rồi Read để xem **khoá / giá trị** là gì (`token`, `access_token`, `password`, `pin`, `jwt`, `session`). Lần theo hàm login: token API trả về được ghi đi đâu.
4. Kiểm `flutter_secure_storage` có được dùng cho token không, và option iOS (`accessibility`, `synchronizable`).
5. Kiểm `.gitignore` cho `key.properties`, `*.jks`, `*.p8`, `service-account*.json`, `.env`.

## Search patterns (gợi ý — dùng Grep tool)

```
String\.fromEnvironment\s*\([^)]*defaultValue:\s*['"][^'"]{12,}
dotenv\.(env\[|get\(|load\()
^\s*-\s*\.env                              # pubspec.yaml assets
(prefs|_prefs|storage|sp)\.setString\s*\(\s*['"]\w*(token|password|pass|pin|jwt|session|secret)
SharedPreferences.*(token|password)
Hive\.openBox\s*\((?![^)]*encryptionCipher)
writeAsString\s*\(.*(token|password)
synchronizable:\s*true
"type"\s*:\s*"service_account"
AAAA[A-Za-z0-9_-]{7}:APA91b                # FCM legacy server key
AuthKey_[A-Z0-9]{10}\.p8
storePassword|keyPassword                  # key.properties
```

## Hardening note (không phải finding) — file tạm chứa dữ liệu nhạy cảm

App chia sẻ / sao chép ảnh biên lai, sao kê, QR (`share_plus` `XFile.fromData`, `pasteboard`, `path_provider` `getTemporaryDirectory`) phải ghi file tạm. OS chỉ dọn khi thiếu chỗ, nên file nằm lại lâu. Nếu app không tự dọn → ghi **hardening note**: "ghi vào thư mục riêng, xoá file cũ ở lần chia sẻ sau và khi đăng xuất". Không gán rule (file nằm trong sandbox của app, không có đường khai thác ngay).

## Examples

### CRITICAL — flag

```yaml
# pubspec.yaml — .env bị đóng gói vào app
flutter:
  assets:
    - assets/images/
    - .env
```
```
# .env
PAYMENT_SECRET_KEY=<secret thật của cổng thanh toán>
```

```dart
// Khoá ký request nằm trong app → ai giải nén app cũng ký được request giả
const _hmacSecret = '9f2c4e...<64 ký tự>';
final apiKey = String.fromEnvironment('OPENAI_KEY', defaultValue: '<key thật>');
```

### HIGH — flag

```dart
// Token đăng nhập lưu SharedPreferences (plist / XML không mã hoá)
Future<void> saveSession(String token) async {
  final prefs = await SharedPreferences.getInstance();
  await prefs.setString('access_token', token);
}
```

### NOT flag

```dart
// firebase_options.dart — cấu hình client, công khai theo thiết kế
static const FirebaseOptions android = FirebaseOptions(
  apiKey: 'AIza...', appId: '1:123:android:abc', messagingSenderId: '123', projectId: 'shop-app',
);
```

```dart
// Token trong Keychain / Keystore, không sang máy khác
const storage = FlutterSecureStorage(
  iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock_this_device),
);
await storage.write(key: 'access_token', value: token);
```

## Fix recommendation

1. Secret server: **xoay key ngay** (coi như đã lộ cho mọi người tải app), chuyển việc gọi API cần secret về backend; app gọi backend bằng token của người dùng.
2. Bỏ `.env` khỏi `flutter: assets:` và bỏ `flutter_dotenv` nếu chỉ để chứa cấu hình; cấu hình công khai dùng hằng số hoặc `--dart-define` (nhớ: `--dart-define` cũng nằm trong binary, không dùng cho secret).
3. Token: `flutter_secure_storage` với `IOSOptions(accessibility: KeychainAccessibility.first_unlock_this_device)` (không iCloud, không theo backup sang máy khác, đọc được khi app chạy nền nhận push). Chuyển dữ liệu từ bản cũ: đọc `SharedPreferences`, ghi Keychain, đọc lại khớp rồi mới xoá chỗ cũ.
4. iOS giữ Keychain sau khi xoá app: lưu một cờ "đã cài" trong `SharedPreferences`; thiếu cờ khi mở app → `deleteAll()` Keychain sót của lần cài trước.
5. Firebase key: hạn chế key theo package name + SHA-1 (Android) và bundle id (iOS) trên Google Cloud Console; bật App Check.

## Cross-references

- `12-broken-access-control` (Dart): `allowBackup` khiến dữ liệu lưu sai chỗ đi theo bản sao lưu.
- `17-verbose-error-debug-mode` (Dart): token bị in ra log.
- `05-slopsquatting` (Dart): gói lưu trữ "an toàn" tên na ná `flutter_secure_storage`.
