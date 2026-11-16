---
id: VERBOSE-ERROR-DEBUG-MODE
severity_max: HIGH
applies_to: dart
overrides: generic/17-verbose-error-debug-mode.md
---

# Verbose Log / Cấu hình debug lọt vào bản phát hành — Dart / Flutter

> Override cho `rules/generic/17-verbose-error-debug-mode.md` khi `primary_language: dart`. Backend Dart (`shelf`, `dart_frog`) trả `e.toString()` / stack trace trong response → áp logic generic.

## Intent (Flutter-specific)

Ở app di động, "debug mode trong production" là: **log** dữ liệu nhạy cảm ở bản release, **cấu hình mạng nới lỏng** để tiện dev mà quên bỏ, và **màn / route debug** không bị loại khỏi bản release.

- `print()` và **`debugPrint()` đều chạy ở bản release** (debugPrint chỉ giãn tốc độ in). Chỉ code trong `if (kDebugMode)` / `assert(...)` mới bị loại khi biên dịch release. Log ra logcat / Console iOS: đọc được qua `adb logcat`, bug report, Xcode Devices, máy root; crash reporter (Crashlytics `log`, Sentry breadcrumb) gửi luôn lên server bên thứ ba.
- Dio `LogInterceptor(requestHeader: true, requestBody: true, responseBody: true)`, `PrettyDioLogger`, `talker_dio_logger` không bọc `kDebugMode` → in header `Authorization`, mật khẩu đăng nhập, toàn bộ response giao dịch.
- Cấu hình mạng: `android:usesCleartextTraffic="true"`, `network_security_config` cho phép cleartext toàn cục hoặc tin `<certificates src="user"/>` ở bản release (cho phép MITM bằng CA người dùng cài), iOS `NSAllowsArbitraryLoads` / `NSExceptionAllowsInsecureHTTPLoads`, `HttpClient..badCertificateCallback = (c, h, p) => true` (tắt kiểm chứng chỉ).
- `android:debuggable="true"` trong manifest chính.

## Khi nào HIGH

- Token / `Authorization` / mật khẩu / OTP / PIN bị log (print, debugPrint, `log()`, logger, Dio LogInterceptor bật header / body) ở đường code chạy release.
- `badCertificateCallback` trả `true` (hoặc `HttpOverrides.global` tương đương) không bọc `kDebugMode`.
- Route / màn debug (xem token, đổi API base URL, bỏ qua đăng nhập, chế độ demo) đăng ký không bọc `kDebugMode` → vào được ở bản release (qua deep link, cử chỉ ẩn).
- Gọi API bằng `http://` cho dữ liệu đăng nhập / giao dịch.

## Khi nào MEDIUM

- Log số tài khoản, số tiền, nội dung giao dịch, payload noti (`message.data`), email / số điện thoại ở bản release.
- `usesCleartextTraffic="true"` / `NSAllowsArbitraryLoads` / `cleartextTrafficPermitted="true"` ở `base-config` trong khi API là https (cửa mở cho request http lọt vào sau này).
- `<certificates src="user"/>` trong `base-config` (không phải `debug-overrides`).
- `android:debuggable="true"` hard-code.
- Chế độ demo / bypass gate chỉ bằng `bool.fromEnvironment('DEMO')` (build nhầm cờ là lọt), không kèm `kDebugMode`.

## Khi nào KHÔNG tạo finding

- Log bọc `if (kDebugMode)`, `assert(() { ...; return true; }())`, hoặc logger có level tắt ở release (`Logger(level: kReleaseMode ? Level.off : Level.debug)`).
- Exception handler log `e.runtimeType` / mã lỗi, trả câu chung cho người dùng.
- Cleartext chỉ cho `localhost` / `10.0.2.2` trong `debug-overrides` hoặc `<domain-config>` dev; ATS exception chỉ cho `localhost`.
- `kDebugMode && bool.fromEnvironment('DEMO')`: bị loại hẳn ở release.
- Thiếu certificate pinning: **hardening note** (không phải finding), nhất là app tài chính; ghi rõ rủi ro khi pin sai (khoá người dùng khi đổi chứng chỉ).

## Cách reasoning

1. Grep `print(`, `debugPrint(`, `log(`, `logger.`, `LogInterceptor`, `PrettyDioLogger`, `FirebaseCrashlytics.instance.log`, `Sentry.addBreadcrumb`. Read ngữ cảnh: có `kDebugMode` / `assert` bao ngoài không? Biến in ra chứa gì (token, header, body, `message.data`)?
2. Đọc `AndroidManifest.xml` (cả `src/main`, không chỉ `src/debug` / `src/profile`), file `res/xml/network_security_config.xml` được trỏ tới, `ios/Runner/Info.plist` mục `NSAppTransportSecurity`.
3. Grep `badCertificateCallback`, `HttpOverrides`, `SecurityContext` và xem có gate theo `kDebugMode` không.
4. Liệt kê route trong router (`GoRoute(path:`, `routes: {`): route debug / wireframe / demo có nằm trong `if (kDebugMode)` không?
5. `src/debug/AndroidManifest.xml`, `src/profile/AndroidManifest.xml` chỉ áp cho bản debug / profile: không flag.

## Search patterns (gợi ý — dùng Grep tool)

```
^\s*(print|debugPrint)\s*\(.*(token|Token|Authorization|password|otp|pin|\.data|headers|body)
LogInterceptor\s*\(|PrettyDioLogger|TalkerDioLogger
(requestHeader|requestBody|responseBody):\s*true
badCertificateCallback\s*=|HttpOverrides\.global
android:usesCleartextTraffic\s*=\s*"true"
cleartextTrafficPermitted\s*=\s*"true"
<certificates\s+src="user"
android:debuggable\s*=\s*"true"
NSAllowsArbitraryLoads|NSExceptionAllowsInsecureHTTPLoads
['"]http://(?!localhost|127\.0\.0\.1|10\.0\.2\.2)
GoRoute\s*\(\s*path:\s*['"]/(debug|dev|wireframes?|demo|test)
bool\.fromEnvironment\s*\(
```

## Examples

### HIGH — flag

```dart
// In token ở mọi bản build
Future<Map> post(String path, Map body) async {
  print('POST $path headers=$_headers body=$body');
  ...
}
```

```dart
// Tắt kiểm chứng chỉ vô điều kiện
class DevHttpOverrides extends HttpOverrides {
  @override
  HttpClient createHttpClient(SecurityContext? c) =>
      super.createHttpClient(c)..badCertificateCallback = (cert, host, port) => true;
}
void main() { HttpOverrides.global = DevHttpOverrides(); runApp(const App()); }
```

### MEDIUM — flag

```xml
<application android:usesCleartextTraffic="true" ...>
```

### NOT flag

```dart
if (kDebugMode) debugPrint('noti foreground → tải lại danh sách');
if (kDebugMode) HttpOverrides.global = DevHttpOverrides();   // chỉ bản debug
final routes = [
  GoRoute(path: '/', builder: ...),
  if (kDebugMode) GoRoute(path: '/wireframes', builder: ...),
];
```

## Fix recommendation

1. Bọc mọi log có dữ liệu người dùng trong `if (kDebugMode)`; không log token / header / body ngay cả trong debug nếu có thể (log `statusCode`, `runtimeType`).
2. Dio: `if (kDebugMode) dio.interceptors.add(LogInterceptor(requestHeader: false, ...))`.
3. Bỏ `usesCleartextTraffic`, `NSAllowsArbitraryLoads`; cleartext / CA người dùng chỉ đặt trong `<debug-overrides>`.
4. Xoá `badCertificateCallback` hoặc gate `kDebugMode`; nếu cần pinning thì pin public key (SPKI) có key dự phòng.
5. Route debug / demo: `if (kDebugMode)` (compile-time), không dùng cờ runtime.

## Cross-references

- `01-hardcoded-secret` (Dart): token lộ qua log tương đương lộ secret.
- `12-broken-access-control` (Dart): route debug mở qua deep link.
