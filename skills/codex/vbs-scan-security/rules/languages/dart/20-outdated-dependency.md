---
id: OUTDATED-DEPENDENCY
severity_max: HIGH
applies_to: dart
overrides: generic/20-outdated-dependency.md
---

# Outdated Dependency — Dart / Flutter (pub.dev)

> Override cho `rules/generic/20-outdated-dependency.md` khi `primary_language: dart`. Repo đa ngôn ngữ (Flutter + backend Node / Go...) thì generic vẫn chạy cho lockfile của ngôn ngữ kia.

## Intent (Flutter-specific)

Package Dart chạy trong tiến trình app, cùng quyền đọc token, Keychain, file của app. Lỗ hổng trong gói HTTP, giải nén, JWT ảnh hưởng thẳng tới dữ liệu khách. vbsec mặc định offline: rule này đối chiếu `pubspec.lock` với danh sách nhỏ dưới đây (đã kiểm trên OSV.dev ecosystem `Pub`), và khuyên chạy `--sca` để tra đầy đủ.

## Cách đọc

1. **`pubspec.lock`** là nguồn version thật: map `packages:`, mỗi key là tên package, field `version: "x.y.z"`, `source: hosted|git|path|sdk`, `dependency: "direct main"|"direct dev"|"transitive"`. Ghi finding ở **dòng `version:` của package đó trong `pubspec.lock`**.
2. Không có `pubspec.lock` (thư viện) → đọc range trong `pubspec.yaml` (`^0.13.0`): range cho phép bản có lỗi mà chưa khoá → MEDIUM, ghi ở `pubspec.yaml`.
3. Bỏ `source: sdk` (`flutter`, `flutter_test`, `flutter_localizations`): đi theo SDK. Phiên bản SDK xem mục dưới.
4. `dependency: "direct dev"` (chỉ lúc build / test) → hạ một mức.

## Static vulnerable list (Pub — subset, đã đối chiếu OSV.dev)

| Package | Vulnerable | Fixed | Advisory | Severity |
|---|---|---|---|---|
| `archive` | `< 3.3.8` | `3.3.8` | CVE-2023-39139 (path traversal khi giải nén), CVE-2023-39137 (giả mạo tên file) | HIGH |
| `dio` | `< 5.0.0` | `5.0.0` | CVE-2021-31402 (CRLF injection qua chuỗi HTTP method) | HIGH |
| `http` | `< 0.13.3` | `0.13.3` | CVE-2020-35669 (header injection) | MEDIUM |
| `jose` | `< 0.3.5+1` | `0.3.5+1` | CVE-2026-34240 (tin JWK trong header khi kiểm chữ ký) | HIGH |

**Dart SDK** (đọc `flutter --version` / `environment: sdk:` trong `pubspec.yaml`; `sdks:` trong `pubspec.lock` chỉ là ràng buộc tối thiểu, không phải bản đang dùng): `< 2.16.0` có CVE-2022-0451 (`HttpClient` gửi lại header `Authorization` khi redirect sang host khác), `< 2.18.2` có CVE-2022-3095 (`Uri` parse dấu `\` sai, lệch host khi so sánh). Chỉ flag MEDIUM khi `environment: sdk:` cho phép bản dưới mức đó **và** repo không có dấu hiệu dùng SDK mới (vd cú pháp Dart 3: `sealed class`, record, pattern).

## Khi nào HIGH

- Package trong bảng trên, version trong `pubspec.lock` thuộc khoảng lỗi, severity cột cuối là HIGH, package là `direct main` hoặc `transitive` được app dùng thật.

## Khi nào MEDIUM

- Package trong bảng với severity MEDIUM, hoặc chỉ có range trong `pubspec.yaml`.
- `dependency_overrides:` ép một package về version trong bảng.
- Package mà pub.dev đánh dấu **discontinued** cho chức năng nhạy cảm (deep link, WebView, lưu trữ), vd `uni_links` (pub.dev gợi ý chuyển `app_links`). Chỉ nêu khi chắc chắn; không đoán.

## Khi nào KHÔNG tạo finding

- Package chỉ "cũ" (có bản major mới hơn) mà không có advisory: không finding. Gợi ý `flutter pub outdated` trong phần Next steps.
- Lockfile lâu không cập nhật: hardening note, không finding.

## Search patterns (gợi ý)

```
^  (archive|dio|http|jose):\s*$            # key package trong pubspec.lock, đọc 5 dòng sau
^\s+version:\s*"([0-9][^"]*)"
^dependency_overrides:
sdk:\s*['"]?>=\s*2\.
```

## Examples

### HIGH — flag

```yaml
# pubspec.lock
  archive:
    dependency: transitive
    description:
      name: archive
      url: "https://pub.dev"
    source: hosted
    version: "3.3.0"        # < 3.3.8: CVE-2023-39139
```

### NOT flag

```yaml
  http:
    dependency: "direct main"
    source: hosted
    version: "1.2.2"        # ≥ 0.13.3
```

## Fix recommendation

1. `flutter pub upgrade <package>` (hoặc nâng ràng buộc trong `pubspec.yaml` rồi `flutter pub get`), chạy lại `flutter test` + thử luồng dùng package đó.
2. Package transitive: nâng package cha; bất đắc dĩ mới dùng `dependency_overrides` và ghi lý do.
3. Tra đầy đủ: `/vbs-scan-security --sca` (OSV.dev ecosystem `Pub`) hoặc `osv-scanner -L pubspec.lock`.
4. Nâng Flutter SDK (kéo theo Dart SDK) định kỳ theo kênh stable.

## Cross-references

- `22-vulnerable-dependency`: tra live qua OSV.dev; trùng package/version thì chỉ giữ rule 22.
- `05-slopsquatting` (Dart): package lạ, git / path dependency.
- `10-path-traversal` (Dart): `archive` cũ + giải nén file ngoài.
