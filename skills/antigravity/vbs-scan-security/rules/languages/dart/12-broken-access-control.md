---
id: BROKEN-ACCESS-CONTROL
severity_max: CRITICAL
applies_to: dart
overrides: generic/12-broken-access-control.md
---

# Broken Access Control — Dart / Flutter (phiên trên máy, backup, deep link)

> Override cho `rules/generic/12-broken-access-control.md` khi `primary_language: dart`. Repo có backend Dart (`shelf`, `dart_frog`, `serverpod`, `alfred`) → với thư mục server, áp nguyên logic generic (route thiếu auth, thiếu kiểm ownership).

## Intent (Flutter-specific)

App client không tự phân quyền dữ liệu: server lọc theo token. "Thiếu kiểm quyền" ở app thường là **kết thúc phiên không sạch** hoặc **cửa vào app mở cho app khác**:

1. **Xoá phiên thiếu RAM**: đăng xuất / 401 chỉ xoá token trong storage rồi về màn đăng nhập; state toàn cục (Provider `ChangeNotifier`, Riverpod, Bloc/Cubit, GetX, singleton) vẫn giữ hồ sơ, danh sách giao dịch, tài khoản của người trước. Máy dùng chung (quầy, cửa hàng): người sau đăng nhập thấy / dùng nhầm dữ liệu người trước.
2. **401 chỉ nhận ở một số verb**: lớp HTTP đổi 401 thành lỗi "hết phiên" ở `get()` nhưng `post()` chỉ ném chuỗi thông báo. Token bị thu hồi (đổi mật khẩu, chủ tài khoản thu quyền) → chính nút Đăng xuất (POST) lỗi → app **kẹt phiên**, dữ liệu vẫn hiện.
3. **Sao lưu mang dữ liệu phiên đi**: Android `android:allowBackup` mặc định `true`; không tắt và không có `dataExtractionRules` / `fullBackupContent` loại `sharedpref`, `file`, `database` → token, cache giao dịch theo bản sao lưu sang máy khác.
4. **Deep link / component exported không dùng**: `<intent-filter>` VIEW + BROWSABLE với scheme riêng, `<activity|service|receiver android:exported="true">` không cần thiết. Flutter mặc định chuyển deep link cho router (`flutter_deeplinking_enabled`), nên app / trang web khác mở được thẳng màn nội bộ (`/transaction/<id>`, `/settings`, màn debug) dù code không có tính năng deep link nào.

## Khi nào HIGH

- Deep link mở được route **thực hiện hành động** không cần xác nhận (chuyển tiền, đổi cài đặt bảo mật, đăng xuất máy khác), hoặc route debug lộ token / đổi API base URL.
- Đăng xuất không xoá cache **trên đĩa** (Hive / sqflite / file) chứa dữ liệu giao dịch của người trước, người sau đăng nhập đọc được.
- `allowBackup` bật (hoặc không khai: mặc định `true`) trong khi token nằm ở `SharedPreferences` / file thường. Tạo finding riêng ở `AndroidManifest.xml`, ngoài finding HARDCODED-SECRET ở chỗ lưu token: hai chỗ sửa khác nhau.

## Khi nào MEDIUM

- Pattern 1: hàm đăng xuất / nhánh 401 không xoá store trong bộ nhớ (chỉ RAM, không ra khỏi tiến trình).
- Pattern 2: lớp HTTP xử lý 401 không đồng nhất giữa các verb, hoặc màn đăng xuất bắt buộc request thành công mới xoá phiên.
- Pattern 3: `allowBackup` không tắt, dữ liệu nhạy cảm (cache giao dịch, hồ sơ) nằm trong sandbox dạng thường; token đã ở Keystore.
- Deep link exported mở được route chỉ-xem (dữ liệu vẫn cần token).

## Khi nào LOW / KHÔNG tạo finding

- Intent-filter custom scheme **có dùng** (code xử lý link, route kiểm định dạng tham số): không finding cho cửa vào; xét tham số ở rule PATH-TRAVERSAL / XSS.
- Chỉ token trong `flutter_secure_storage` (Keystore, khoá không xuất được) mà `allowBackup` bật: hardening note.
- Coi **mọi** 401 là hết phiên kể cả request đăng nhập: không phải lỗ hổng (chỉ mất câu báo "sai mật khẩu" vì nhiều API trả 401 cho cả hai). Khi gợi ý sửa pattern 2, chỉ coi 401 là hết phiên **khi request có gửi token**.
- App không có khoá đăng nhập sau N lần sai, không có PIN / sinh trắc: không phải finding (chống dò là việc server). Thiếu lớp che app switcher (`AppLifecycleState.inactive` / `FLAG_SECURE`) cho app hiện số dư: **hardening note**.

## Cách reasoning

1. Tìm mọi điểm kết thúc phiên: `logout`, `signOut`, nhánh `401` / `Unauthorized` / `SessionExpired`, `onResume` kiểm phiên, interceptor `onError` của Dio. Liệt kê store toàn cục (`ChangeNotifier`, `StateNotifier`, `Cubit`, `GetxController`, singleton có field dữ liệu người dùng). Mỗi nhánh có gọi một hàm xoá chung (`clearSession`) xoá token + mọi store + header HTTP + cache đĩa không?
2. Đọc lớp HTTP (`http.Client` wrapper, Dio interceptor): 401 được nhận diện ở verb nào? `post` / `put` / `delete` có cùng nhánh không? Đăng xuất gặp 401 có xoá cục bộ không?
3. Đọc `android/app/src/main/AndroidManifest.xml`: `allowBackup`, `fullBackupContent`, `dataExtractionRules` (đọc file XML được trỏ tới), mọi `exported="true"`, mọi `<intent-filter>` có `android.intent.action.VIEW`. iOS: `CFBundleURLTypes`, `com.apple.developer.associated-domains` trong entitlements.
4. Đối chiếu cửa vào với code: có `app_links` / `uni_links` / `getInitialLink` / route nào dùng scheme đó không? Router có `redirect` chặn khi chưa đăng nhập, route debug có bọc `kDebugMode` không?

## Search patterns (gợi ý — dùng Grep tool)

```
(logout|signOut|clearSession|onUnauthorized)\s*\(
(==|case)\s*['"]?401|statusCode\s*==\s*401|UnauthorizedException|SessionExpired
(prefs|storage|storageHelper)\.(clear|remove|delete)\s*\(
extends\s+(ChangeNotifier|StateNotifier|Cubit|Bloc|GetxController)
android:allowBackup|android:fullBackupContent|android:dataExtractionRules
android:exported\s*=\s*"true"
android\.intent\.action\.VIEW|android:scheme=
flutter_deeplinking_enabled
CFBundleURLSchemes|associated-domains
```

## Examples

### MEDIUM — flag

```dart
// 401 chỉ xoá storage; UserModel / TransactionModel vẫn giữ dữ liệu người trước
} catch (e) {
  if (e == '401') {
    await storage.clear();
    context.go('/login');
  }
}
```

```dart
// get() nhận 401, post() thì không → token bị thu hồi thì Đăng xuất (POST) kẹt
Future get(String path) async { ... if (res.statusCode == 401) throw SessionExpired(); }
Future post(String path, Map body) async { ... if (res.statusCode != 200) throw json['message']; }
```

```xml
<!-- Deep link không ai dùng, router vẫn nhận: app khác mở thẳng màn nội bộ -->
<intent-filter>
  <action android:name="android.intent.action.VIEW" />
  <category android:name="android.intent.category.BROWSABLE" />
  <data android:scheme="shopapp" />
</intent-filter>
```

### NOT flag

```dart
// Một hàm xoá phiên, gọi ở đăng xuất và mọi nhánh 401
Future<void> clearSession() async {
  _token = null; _profile = null; _accounts = [];
  http.setToken('');
  await secureStorage.delete(key: 'access_token');
  notifyListeners();
}
// Lớp HTTP: có token mà bị 401 → hết phiên; đăng nhập (chưa có token) giữ câu báo của server
if (hasToken && res.statusCode == 401) throw SessionExpired();
```

## Fix recommendation

1. Gom về **một** `clearSession()` xoá token (RAM + Keychain), mọi store, header HTTP, cache đĩa, file tạm; gọi ở đăng xuất, 401 lúc mở app, 401 lúc quay lại app, 401 trong request nền. Viết test: sau `clearSession()` mọi getter rỗng.
2. Lớp HTTP ném **một** lỗi hết phiên cho mọi verb khi request có token mà nhận 401. Đăng xuất gặp lỗi đó → xoá cục bộ, về màn đăng nhập (server đã không còn nhận token).
3. `AndroidManifest.xml`: `android:allowBackup="false"`, `android:fullBackupContent="false"`, `android:dataExtractionRules="@xml/data_extraction_rules"` loại `root`, `file`, `database`, `sharedpref`, `external` ở cả `<cloud-backup>` và `<device-transfer>`.
4. Bỏ intent-filter / URL type không dùng; nếu cần deep link, khai đúng host + path, kiểm tham số theo định dạng, route nhạy cảm yêu cầu xác nhận.

## Cross-references

- `01-hardcoded-secret` (Dart): token lưu sai chỗ.
- `10-path-traversal` (Dart): tham số deep link / push ghép vào đường dẫn router.
- `17-verbose-error-debug-mode` (Dart): route debug không bọc `kDebugMode`.
