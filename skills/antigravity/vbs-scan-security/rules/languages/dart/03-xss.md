---
id: XSS
severity_max: HIGH
applies_to: dart
overrides: generic/03-xss.md
---

# XSS / Mở URL không tin cậy — Dart / Flutter

> Override cho `rules/generic/03-xss.md` khi `primary_language: dart`.

## Intent (Flutter-specific)

Widget Flutter (`Text`, `RichText`) không thông dịch HTML, nên XSS kiểu web chỉ xuất hiện ở ba chỗ:

1. **WebView** (`webview_flutter`, `flutter_inappwebview`): nạp URL hoặc HTML lấy từ dữ liệu ngoài. Trang lạ chạy JavaScript trong WebView của app; nếu app mở `JavaScriptChannel` / `addJavaScriptHandler`, tiêm token vào header / cookie / `runJavaScript`, trang đó gọi được hàm native hoặc lấy token.
2. **Mở URL** (`url_launcher` `launchUrl` / `launch`, `Navigator` sang màn WebView) với URL từ payload push, deep link, response API, nội dung người dùng khác nhập: mở trang lừa đảo trong ngữ cảnh "app ngân hàng gửi", hoặc scheme nguy hiểm (`javascript:`, `intent:`, `file:`, `tel:`/`sms:` tự quay số).
3. **Flutter Web**: `dart:html` / `package:web` `innerHtml`, `setInnerHtml(..., treeSanitizer: NodeTreeSanitizer.trusted)`, `HtmlElementView`. Áp như XSS DOM của generic.

Render HTML bằng `flutter_html` / `flutter_widget_from_html` không chạy script, nhưng `onLinkTap` → `launchUrl(url)` quay về loại 2.

## Khi nào HIGH

- WebView nạp URL / HTML từ L1 (push, deep link, clipboard, nội dung người dùng khác) với `JavaScriptMode.unrestricted` và có `addJavaScriptChannel` / `addJavaScriptHandler` / tiêm token.
- `loadHtmlString` / `loadData` với HTML ghép từ dữ liệu ngoài, JS bật.
- Flutter Web: `innerHtml =` / `NodeTreeSanitizer.trusted` với dữ liệu L1.

## Khi nào MEDIUM

- `launchUrl(Uri.parse(x))` với `x` từ payload push / deep link / response API mà không whitelist scheme + host.
- WebView nạp URL ngoài không whitelist, JS bật, không có channel / token (rủi ro lừa đảo trong khung app).
- `onNavigationRequest` luôn trả `NavigationDecision.navigate` cho WebView đang có channel.

## Khi nào KHÔNG tạo finding

- URL là hằng số trong code (trang điều khoản, hỗ trợ, `app-settings:`).
- Đã whitelist: `uri.scheme == 'https' && allowedHosts.contains(uri.host)` trước khi mở / nạp.
- Mở bằng trình duyệt ngoài (`LaunchMode.externalApplication`) với URL https đã whitelist host.
- Màn quên mật khẩu mở `SFSafariViewController` / Custom Tabs (`LaunchMode.inAppBrowserView`) thay vì WebView: app không đọc được mật khẩu người dùng gõ. Đây là cách đúng.

## Cách reasoning

1. Grep `WebViewController`, `InAppWebView`, `loadRequest`, `loadHtmlString`, `launchUrl`, `launch(`, `canLaunchUrl`, `innerHtml`.
2. Lần URL / HTML về nguồn (xem `data-flow-classification.md` mục "App di động"). Response API của chính backend là L2: chỉ thành L1 khi field đó do người dùng khác nhập (mô tả, ghi chú, link người dùng dán).
3. Kiểm whitelist scheme + host **sau** `Uri.parse` (không kiểm bằng `startsWith('https://shop.vn')`: `https://shop.vn.evil.com` vẫn qua; check viết sai vẫn là finding).
4. WebView: liệt kê channel / handler và thứ chúng làm (đọc token? gọi API? mở camera?).

## Search patterns (gợi ý — dùng Grep tool)

```
(launchUrl|launchUrlString|launch)\s*\(\s*Uri\.parse\s*\(\s*(?!['"])
loadRequest\s*\(\s*Uri\.parse\s*\(\s*(?!['"])
loadHtmlString\s*\(|loadData\s*\(
setJavaScriptMode\s*\(\s*JavaScriptMode\.unrestricted
addJavaScriptChannel|addJavaScriptHandler|evaluateJavascript|runJavaScript
NavigationDecision\.navigate
startsWith\s*\(\s*['"]https?://           # whitelist viết sai
\.innerHtml\s*=|setInnerHtml|NodeTreeSanitizer\.trusted
```

## Examples

### HIGH — flag

```dart
// URL từ deep link /help?u=... nạp vào WebView có channel trả token
final controller = WebViewController()
  ..setJavaScriptMode(JavaScriptMode.unrestricted)
  ..addJavaScriptChannel('App', onMessageReceived: (m) {
    controller.runJavaScript('onToken("${session.token}")');
  })
  ..loadRequest(Uri.parse(widget.url));
```

### MEDIUM — flag

```dart
// Link trong payload push mở thẳng
FirebaseMessaging.onMessageOpenedApp.listen((m) {
  final link = m.data['link'];
  if (link != null) launchUrl(Uri.parse(link));
});
```

### NOT flag

```dart
const _allowed = {'example.com', 'help.example.com'};
final uri = Uri.tryParse(link ?? '');
if (uri != null && uri.scheme == 'https' && _allowed.contains(uri.host)) {
  await launchUrl(uri, mode: LaunchMode.externalApplication);
}
```

## Fix recommendation

1. Whitelist scheme `https` + host cố định sau `Uri.tryParse`; không khớp thì bỏ.
2. WebView chỉ nạp host của mình; `onNavigationRequest` chặn host lạ (mở ra trình duyệt ngoài); không tiêm token vào trang, không mở channel trả dữ liệu nhạy cảm.
3. Trang đăng nhập / quên mật khẩu bên thứ ba: `LaunchMode.inAppBrowserView` hoặc trình duyệt ngoài, không WebView.
4. Flutter Web: dùng `Text` hoặc sanitize trước khi gán HTML.

## Cross-references

- `10-path-traversal` (Dart): cùng nguồn push / deep link ghép vào route.
- `12-broken-access-control` (Dart): deep link exported là cửa vào của chuỗi lỗi này.
