---
id: OUTDATED-DEPENDENCY
severity_max: HIGH
applies_to: all
---

# Outdated Dependency (Known CVE)

## Intent

Dependency cũ có **CVE đã công bố** → exploit code có sẵn trên GitHub, hacker chỉ việc copy-paste. Vibe coder cài package từ tutorial 2 năm trước, không bao giờ update → mang theo cả tá lỗ hổng (log4shell, prototype pollution, RCE, SSRF...).

vbsec mặc định chạy **offline** (không fetch CVE DB), nên rule này:
1. Flag một số package + version **well-known vulnerable** (static list dưới)
2. Khuyến nghị user tự chạy `npm audit` / `pip-audit` / `govulncheck` / `composer audit`

**v0.7+**: nếu user truyền flag `--sca`, vbsec tra cứu **live** qua OSV.dev API — xem [`22-vulnerable-dependency`](22-vulnerable-dependency.md) + [`../../references/dependency-scan.md`](../../references/dependency-scan.md). Rule 22 có `cve_id`/`fixed_version` chính xác hơn static list dưới; khi cùng package/version match cả 2 rule, chỉ giữ finding của rule 22. Rule này (20) vẫn là fallback bắt buộc khi không có `--sca` hoặc network không khả dụng.

## Khi nào HIGH

- Package nằm trong static list dưới với version vulnerable
- `package-lock.json` / `requirements.txt` / `go.sum` / `composer.lock` show version cụ thể có CVE
- Lockfile cũ > 12 tháng không update (signal có nhiều CVE chưa patch)

## Khi nào MEDIUM (giảm cấp)

- Package outdated nhưng chưa có CVE biết đến
- Dev dependency (chỉ chạy lúc build, không production)
- Version bracket cho phép latest (`^1.2.3`) nhưng chưa run `npm install`

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Đọc lockfile**: `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `requirements.txt`, `Pipfile.lock`, `poetry.lock`, `go.sum`, `composer.lock`, `Gemfile.lock`
2. **Check static list** dưới đây — đây là **subset nhỏ** của CVE nổi tiếng, KHÔNG đầy đủ
3. **Đếm tuổi lockfile**: `git log -1 --format=%ai <lockfile>` — quá 12 tháng → warn
4. **Khuyến nghị user**: chạy audit tool thật để có dữ liệu đầy đủ

## Static vulnerable list (subset — well-known CVEs)

### JavaScript / npm

| Package | Vulnerable | Fixed | CVE |
|---|---|---|---|
| `lodash` | `< 4.17.21` | `4.17.21` | CVE-2021-23337 (command injection in template) |
| `jquery` | `< 3.5.0` | `3.5.0` | CVE-2020-11022/11023 (XSS) |
| `axios` | `< 0.21.1` | `0.21.1` | CVE-2020-28168 (SSRF) |
| `axios` | `< 1.6.0` | `1.6.0` | CVE-2023-45857 (CSRF token leak) |
| `node-fetch` | `< 2.6.7` | `2.6.7` | CVE-2022-0235 (info leak) |
| `minimist` | `< 1.2.6` | `1.2.6` | CVE-2021-44906 (prototype pollution) |
| `ws` | `< 7.4.6` | `7.4.6` | CVE-2021-32640 (ReDoS) |
| `tar` | `< 6.1.9` | `6.1.9` | CVE-2021-37701 (arbitrary file write) |
| `serialize-javascript` | `< 3.1.0` | `3.1.0` | CVE-2020-7660 (XSS) |
| `handlebars` | `< 4.7.7` | `4.7.7` | CVE-2021-23369 (RCE) |
| `next` | `< 13.5.1` | `13.5.1` | CVE-2023-46298 (cache poisoning) |
| `express` | `< 4.17.3` | `4.17.3` | CVE-2022-24999 (qs prototype pollution) |

### Python / pip

| Package | Vulnerable | Fixed | CVE |
|---|---|---|---|
| `django` | `< 3.2.25` / `< 4.2.11` | latest | various |
| `flask` | `< 2.2.5` | `2.2.5` | CVE-2023-30861 (session cookie) |
| `requests` | `< 2.31.0` | `2.31.0` | CVE-2023-32681 (proxy auth leak) |
| `pillow` | `< 10.2.0` | `10.2.0` | multiple buffer overflow |
| `pyyaml` | `< 5.4` | `5.4` | CVE-2020-14343 (arbitrary code) |
| `urllib3` | `< 1.26.18` | `1.26.18` | CVE-2023-45803 (request smuggling) |
| `jinja2` | `< 3.1.3` | `3.1.3` | CVE-2024-22195 (XSS) |
| `werkzeug` | `< 3.0.1` | `3.0.1` | CVE-2023-46136 (DoS) |
| `cryptography` | `< 41.0.6` | `41.0.6` | various |

### Java / Maven

| Package | Vulnerable | Fixed | CVE |
|---|---|---|---|
| `log4j-core` | `< 2.17.1` | `2.17.1` | CVE-2021-44228 (Log4Shell — CRITICAL RCE) |
| `spring-core` | `< 5.3.20` | `5.3.20` | CVE-2022-22965 (Spring4Shell RCE) |
| `jackson-databind` | `< 2.13.4` | `2.13.4` | multiple deserialization RCE |
| `fastjson` | `< 1.2.83` | `1.2.83` | CVE-2022-25845 (RCE) |

### PHP / Composer

| Package | Vulnerable | Fixed | CVE |
|---|---|---|---|
| `laravel/framework` | `< 9.52.16` | `9.52.16` | CVE-2024-29291 (SQL injection) |
| `symfony/http-kernel` | `< 5.4.20` | `5.4.20` | various |
| `guzzlehttp/guzzle` | `< 7.4.5` | `7.4.5` | CVE-2022-31091 (cookie leak) |

### Go / Ruby

| Package | Vulnerable | Fixed | CVE |
|---|---|---|---|
| `github.com/gin-gonic/gin` | `< 1.9.0` | `1.9.0` | CVE-2023-26125 (path traversal) |
| `golang.org/x/net` | `< 0.17.0` | `0.17.0` | CVE-2023-39325 (HTTP/2 DoS) |
| `rack` (Ruby) | `< 2.2.6.4` | `2.2.6.4` | CVE-2023-27530 (DoS) |
| `nokogiri` | `< 1.13.10` | `1.13.10` | multiple |

## Search patterns

### Lockfiles

```
# Node.js
"version":\s*"([\d.]+)"     # trong package-lock.json
# Check "name" + version

# Python
^([a-zA-Z0-9_\-]+)==(.+)$   # trong requirements.txt
^([a-zA-Z0-9_\-]+)\s*=\s*"([^"]+)"  # trong Pipfile

# Go
go\.mod / go.sum

# PHP
"version":\s*"v?([\d.]+)"   # composer.lock
```

## Examples

### HIGH — flag

```json
// package-lock.json — lodash 4.17.20 vulnerable
{
  "node_modules/lodash": {
    "version": "4.17.20"    // CVE-2021-23337
  }
}
```

```
# requirements.txt
django==3.2.5           # nhiều CVE chưa patch
pillow==8.0.0           # buffer overflow CVEs
requests==2.25.1        # CVE-2023-32681
pyyaml==5.3             # CVE-2020-14343
```

```xml
<!-- pom.xml — Log4Shell -->
<dependency>
    <groupId>org.apache.logging.log4j</groupId>
    <artifactId>log4j-core</artifactId>
    <version>2.14.1</version>   <!-- RCE CRITICAL -->
</dependency>
```

### NOT critical

```json
// Latest patched version
{
  "node_modules/lodash": { "version": "4.17.21" }
}
```

```
# requirements.txt — pinned to latest patched
django==4.2.11
requests==2.31.0
```

## Fix recommendation

1. **Chạy audit tool — ƯU TIÊN trước mọi việc:**
   ```bash
   # Node
   npm audit fix
   yarn audit
   pnpm audit

   # Python
   pip install pip-audit && pip-audit
   safety check

   # Go
   go install golang.org/x/vuln/cmd/govulncheck@latest
   govulncheck ./...

   # PHP
   composer audit

   # Ruby
   bundle audit check --update

   # Java
   mvn org.owasp:dependency-check-maven:check
   ```
2. **Update** package có CVE → version đã fix:
   ```bash
   npm update lodash@^4.17.21
   pip install --upgrade requests
   ```
3. **Dependabot / Renovate**: bật auto-PR khi có security update.
4. **SBOM** (Software Bill of Materials): generate với `syft`, `cyclonedx-bom` để track dependency.
5. **Pin version đầy đủ** trong lockfile, commit lockfile.
6. **Loại bỏ dependency không cần**: mỗi package thêm = attack surface thêm.
7. **Re-audit định kỳ** (weekly trong CI).

## Cross-references

- **Lưu ý**: list trên KHÔNG đầy đủ — chỉ là well-known CVE. Phải chạy audit tool để có data thật.
- `22-vulnerable-dependency`: bản "live" của rule này, dùng khi có `--sca`. Không double-report cùng package/version ở cả 2 rule.
- Cross-check với `01-hardcoded-secret`: package cũ có thể leak qua telemetry
- Cross-check với `17-verbose-error-debug-mode`: error stack reveal version → CVE lookup dễ
