---
id: SLOPSQUATTING
severity_max: CRITICAL
applies_to: dart
overrides: generic/05-slopsquatting.md
---

# Slopsquatting / Package lạ — Dart / Flutter (pub.dev)

> Override cho `rules/generic/05-slopsquatting.md` khi `primary_language: dart`.

## Intent (Flutter-specific)

AI hay bịa tên package Flutter nghe hợp lý (`flutter_secure_storage_plus`, `firebase_messaging_helper`, `dio_retry_pro`). Kẻ xấu đăng ký tên đó trên pub.dev. Pub không có post-install script như npm, nhưng vẫn nguy hiểm:

- Code package chạy **trong tiến trình app**: đọc token, Keychain, gửi payload noti, số dư ra ngoài.
- **Build hook** (`hook/build.dart`, native assets) và builder của `build_runner` chạy trên máy dev / CI lúc build: đọc `~/.ssh`, `key.properties`, keystore ký app.
- Plugin có mã native (`android/`, `ios/`) xin thêm quyền hệ điều hành qua manifest merge.

## Khi nào CRITICAL

- Tên trong `pubspec.yaml` gần giống package phổ biến (khoảng cách ≤ 2 ký tự: `flutter_secure_storge`, `shared_preference`, `firebase_mesaging`, `go_routers`) và không có trong `pubspec.lock` (chưa cài), hoặc có trong lock nhưng không thấy nguồn nào nhắc tới.
- Package "bản nâng cấp" của gói nhạy cảm (lưu trữ bảo mật, mã hoá, HTTP, thanh toán, noti) với hậu tố lạ, publisher không xác minh.

## Khi nào MEDIUM

- `git:` dependency không khoá `ref:` theo commit SHA (nhánh có thể bị đẩy code mới), hoặc trỏ tới fork cá nhân của gói nhạy cảm.
- `hosted:` trỏ tới server package lạ (không phải `https://pub.dev` hay server nội bộ của công ty).
- `dependency_overrides:` thay gói nhạy cảm bằng `path:` / `git:` không giải thích.

## Khi nào KHÔNG tạo finding

- Gói hậu tố `_plus` của **Flutter Community** (`share_plus`, `package_info_plus`, `device_info_plus`, `connectivity_plus`, `sensors_plus`, `battery_plus`, `network_info_plus`, `android_alarm_manager_plus`): bản kế nhiệm chính thức của plugin Flutter team cũ, publisher `fluttercommunity.dev`.
- Gói của publisher đã xác minh: `dart.dev`, `flutter.dev`, `google.dev`, `firebase.google.com`, `tools.dart.dev`, `fluttercommunity.dev`.
- Gói phổ biến tên đúng chính tả và có trong `pubspec.lock` với `source: hosted`, `url: "https://pub.dev"`.
- `path:` trỏ vào package nội bộ trong cùng repo (monorepo, `packages/`).

## Cách reasoning

1. Đọc `dependencies`, `dev_dependencies`, `dependency_overrides` trong mọi `pubspec.yaml` (kể cả `packages/*/pubspec.yaml`); đối chiếu `pubspec.lock` (`source`, `description.url`).
2. Loại import nội bộ (`package:<tên app>/`), `dart:` và gói SDK (`flutter`, `flutter_test`, `flutter_localizations`, `integration_test`).
3. Với mỗi tên còn lại: giống tên phổ biến không? hậu tố lạ (`_pro`, `_helper`, `_utils`, `_secure`, `_v2`) trên gói nhạy cảm? publisher? Scanner không tự lên mạng: ghi finding kèm hướng dẫn kiểm `https://pub.dev/packages/<tên>` (publisher, likes, ngày đăng, repo).
4. Import trong code (`import 'package:<tên>/`) không có trong `pubspec.yaml` → thiếu khai báo (lỗi build), không phải slopsquatting; nhưng nếu AI sắp thêm tên đó vào thì kiểm trước khi `pub add`.

## Search patterns (gợi ý — dùng Grep tool)

```
^\s{2}[a-z0-9_]+:\s*(\^|>=|any|\d)          # dependency trong pubspec.yaml
^\s+git:|^\s+url:\s*(?!"?https://pub\.dev)  # git / hosted lạ
^dependency_overrides:
import\s+['"]package:([a-z0-9_]+)/
_(pro|helper|helpers|utils|secure|v2|plus_plus)\b
```

## Examples

### CRITICAL — flag

```yaml
dependencies:
  flutter_secure_storge: ^9.0.0     # sai chính tả flutter_secure_storage, không có trong lock
```

### MEDIUM — flag

```yaml
dependencies:
  dio:
    git:
      url: https://github.com/someone/dio.git   # fork cá nhân, không khoá ref
```

### NOT flag

```yaml
dependencies:
  share_plus: ^10.1.4           # Flutter Community, kế nhiệm plugin `share`
  firebase_messaging: ^15.2.0   # firebase.google.com
```

## Fix recommendation

1. Kiểm package trên pub.dev trước khi thêm (publisher xác minh, repo, số likes, lịch sử); sai thì xoá, dùng gói đúng tên.
2. Git dependency: khoá `ref:` bằng commit SHA, ưu tiên bản hosted chính thức.
3. Commit `pubspec.lock` cho app (không cho thư viện) để version cố định giữa các máy và CI.

## Cross-references

- `20-outdated-dependency` (Dart): version có CVE.
- `22-vulnerable-dependency`: tra OSV.dev ecosystem `Pub` với `--sca`.
