---
id: PATH-TRAVERSAL
severity_max: HIGH
applies_to: dart
overrides: generic/10-path-traversal.md
---

# Path Traversal — Dart / Flutter (đường dẫn điều hướng và đường dẫn file)

> Override cho `rules/generic/10-path-traversal.md` khi `primary_language: dart`. Backend Dart (`shelf`, `shelf_static`, `dart_frog`) đọc file theo tham số request → áp nguyên logic generic (`File(p.join(base, req.params['f']))` thiếu kiểm prefix sau `p.canonicalize`).

## Intent (Flutter-specific)

Ở app Flutter có hai loại "đường dẫn" nhận dữ liệu ngoài:

1. **Đường dẫn điều hướng**: giá trị từ payload push (`message.data['transaction_id']`) hay deep link ghép thẳng vào `context.go('/transaction/$id')`, `GoRouter.go`, `Navigator.pushNamed('/x/$id')`. Giá trị chứa `/`, `?`, `#`, `..` đổi luôn route đích: mở màn khác, chèn query, nhảy vào route debug. Nguồn push là server của mình, nhưng ai có FCM server key / backend bị lạm dụng cũng gửi được; nguồn deep link thì app hay trang web bất kỳ gửi được.
2. **Đường dẫn file**: tên file lấy từ dữ liệu ngoài ghép vào `File('${dir.path}/$name')`: entry trong zip người dùng chọn (ZipSlip), tên file trong payload / response, file chia sẻ từ app khác (`receive_sharing_intent`). `../` ghi đè file của app (cấu hình, database, token fallback).

## Khi nào HIGH

- ZipSlip: giải nén archive từ nguồn ngoài (file người dùng chọn, tải từ URL ngoài) bằng `package:archive`, ghi `File('${out}/${file.name}')` mà không kiểm đường dẫn sau chuẩn hoá nằm trong thư mục đích. `archive` < 3.3.8 còn có CVE path traversal riêng (xem OUTDATED-DEPENDENCY).
- Tên file từ deep link / file chia sẻ / clipboard ghép vào đường dẫn ghi hoặc xoá file.

## Khi nào MEDIUM

- Tham số deep link (`state.pathParameters`, `state.uri.queryParameters`, `uriLinkStream`) ghép vào đường dẫn điều hướng khác mà không lọc định dạng.
- Tên file từ response API ghép vào đường dẫn ghi file.

## Khi nào LOW

- Giá trị từ **payload push** ghép vào đường dẫn điều hướng mà không lọc định dạng (nguồn là server của mình, cần lạm dụng quyền gửi). Vẫn là finding: thiếu kiểm định dạng ở cửa vào.

## Khi nào KHÔNG tạo finding

- Giá trị đã lọc theo **đúng định dạng id trong tài liệu API** trước khi ghép: `RegExp(r'^[0-9a-f-]{36}$')`, `int.tryParse` (khi id là số), `Uri.encodeComponent`. Chú ý lọc phải khớp định dạng thật: đoán "chỉ số" trong khi API dùng UUID làm hỏng tính năng, gợi ý sửa phải dẫn tài liệu API.
- Truyền qua `extra:` / `pathParameters:` của `goNamed` (go_router tự encode) thay vì nối chuỗi.
- Tên file do app đặt (hằng số, `DateTime.now()`, `Uuid().v4()`).
- Check viết sai (`contains('..')` trước khi decode, `startsWith(base)` không có `/` cuối) **vẫn là finding**, hạ severity, nêu cách bypass.

## Cách reasoning

1. Grep sink điều hướng (`context.go(`, `context.push(`, `.go('/`, `pushNamed(`) có nội suy `$` / `+`; grep sink file (`File(`, `Directory(`, `p.join(`, `writeAsBytes`, `extractArchiveToDisk`, `ZipDecoder`).
2. Lần ngược biến về nguồn: `RemoteMessage.data`, `getInitialMessage`, `onMessageOpenedApp`, `NotificationResponse.payload`, `GoRouterState`, `app_links`, `receive_sharing_intent`, `FilePicker`, `ArchiveFile.name`.
3. Có lọc định dạng / chuẩn hoá ở **mọi** đường vào không (thường có 2-3 đường: app đang mở, app ở nền, app mở từ noti lúc tắt)? Một đường lọc, đường khác không → finding ở đường thiếu.
4. ZipSlip: có `p.normalize(p.join(out, name))` rồi `p.isWithin(out, dest)` không?

## Search patterns (gợi ý — dùng Grep tool)

```
context\.(go|push|replace|pushReplacement)\s*\(\s*['"][^'"]*\$
\.go\s*\(\s*['"]/[^'"]*\$\{?\w*(id|Id|data)
pushNamed\s*\(\s*['"][^'"]*\$
message\.data\[\s*['"]\w+['"]\s*\]
(getInitialMessage|onMessageOpenedApp|onDidReceiveNotificationResponse)
state\.(pathParameters|uri\.queryParameters|queryParameters)
File\s*\(\s*['"]\$\{?\w+(\.path)?\}?/\$
ZipDecoder\(\)\.decode|extractArchiveToDisk|extractFileToDisk|ArchiveFile
```

## Examples

### HIGH — flag

```dart
// ZipSlip: entry "../../shared_prefs/FlutterSharedPreferences.xml" ghi đè dữ liệu app
final archive = ZipDecoder().decodeBytes(await picked.readAsBytes());
for (final f in archive.files) {
  final out = File('${dir.path}/${f.name}');
  await out.create(recursive: true);
  await out.writeAsBytes(f.content as List<int>);
}
```

### LOW — flag

```dart
// transaction_id từ push ghép thẳng vào route
FirebaseMessaging.onMessageOpenedApp.listen((m) {
  router.go('/transaction/${m.data['transaction_id']}');
});
```

### NOT flag

```dart
// Lọc theo định dạng id của API (UUID hoặc số cũ), sai thì không mở màn nào
String? safeTransactionId(Object? raw) =>
    raw is String && RegExp(r'^[A-Za-z0-9-]{1,64}$').hasMatch(raw) ? raw : null;

final id = safeTransactionId(message.data['transaction_id']);
if (id != null && mounted) context.go('/transaction/$id');
```

```dart
final dest = p.normalize(p.join(dir.path, f.name));
if (!p.isWithin(dir.path, dest)) continue;   // bỏ entry thoát khỏi thư mục đích
```

## Fix recommendation

1. Một hàm lọc dùng chung cho mọi đường vào (push lúc mở, lúc nền, lúc tắt; deep link), lọc theo định dạng id **trong tài liệu API**; không khớp thì bỏ, không điều hướng.
2. Dùng `goNamed(..., pathParameters: {'id': id})` thay vì nối chuỗi.
3. Listener mở màn từ noti (`eventBus.on<...>().listen`, `onMessageOpenedApp.listen`) giữ `StreamSubscription`, `cancel()` ở `dispose()`, kiểm `mounted` sau `await`.
4. Giải nén: `p.isWithin` sau `p.normalize`, bỏ entry symlink, nâng `archive` ≥ 3.3.8.
5. Tên file ngoài: chỉ giữ `[A-Za-z0-9._-]`, hoặc tự đặt tên.

## Cross-references

- `12-broken-access-control` (Dart): deep link exported không dùng.
- `03-xss` (Dart): giá trị push / deep link dùng làm URL mở WebView / trình duyệt.
- `20-outdated-dependency` (Dart): `archive` < 3.3.8.
