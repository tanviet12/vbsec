# Dependency Scan — Live SCA via OSV.dev (`--sca`)

Hướng dẫn parse dependency manifest và tra cứu CVE live qua [OSV.dev](https://osv.dev) API cho rule [`22-vulnerable-dependency`](../rules/generic/22-vulnerable-dependency.md).

> Được gọi từ [`../SKILL.md`](../SKILL.md) **Step 4b** khi user truyền flag `--sca`. Chạy sau khi Step 2 đã detect `$PRIMARY_LANG` (và multi-lang nếu có).
>
> **Đây là 1 trong 2 chỗ duy nhất trong skill được phép gọi network** (chỗ còn lại là `gh pr diff` cho scope `pr id`). Mọi request là read-only GET/POST tới `api.osv.dev`, không gửi code hay nội dung file đi — chỉ gửi package name + version.

## Bước 1 — Đọc manifest theo ecosystem

vbsec hiện chuyên sâu 5 ngôn ngữ — mỗi ngôn ngữ map sang đúng 1 [OSV ecosystem name](https://ossf.github.io/osv-schema/#affectedpackage-field):

| Lang (vbsec) | OSV ecosystem | Manifest/lockfile đọc | Cách trích PackageId + Version (illustrative) |
|---|---|---|---|
| `dotnet` | `NuGet` | `*.csproj` (`<PackageReference Include="X" Version="Y" />`), `packages.lock.json` | Regex `Include="([^"]+)"\s+Version="([^"]+)"` trong `.csproj`; hoặc field `resolved` trong `packages.lock.json` (chính xác hơn — version đã resolve) |
| `go` | `Go` | `go.mod` (`require X vY`) | Mỗi dòng trong block `require (...)` hoặc `require X vY` đơn dòng |
| `typescript` | `npm` | `package-lock.json` (lockfile v2/v3, object `packages`) | Ưu tiên `package-lock.json` hơn `package.json` — có version đã resolve thật, không phải range (`^1.2.3`). Walk `packages["node_modules/<name>"].version` |
| `php` | `Packagist` | `composer.lock` (`packages[].name` + `.version`) | JSON parse trực tiếp — `composer.lock` luôn có version cụ thể |
| `python` | `PyPI` | `requirements.txt` (`pkg==version`), `poetry.lock`, `Pipfile.lock` | `requirements.txt`: split theo `==` (bỏ qua dòng dùng `>=`/`~=` — không có version cụ thể để query); `poetry.lock`/`Pipfile.lock`: parse TOML/JSON lấy `name` + `version` |

**Nguyên tắc đọc:** Dùng Read tool, KHÔNG chạy lệnh `jq`/parser thật — đây là illustrative pattern, agent tự parse bằng reasoning giống cách đọc code khác. Nếu 1 repo có nhiều manifest cùng ecosystem (vd nhiều `.csproj` trong monorepo), gộp tất cả package tìm được, dedup theo `(name, version)`.

Nếu không tìm thấy manifest nào cho `$PRIMARY_LANG` (hoặc bất kỳ lang nào trong multi-lang repo) → note `{msg_sca_no_manifest}`, skip ecosystem đó, không lỗi toàn bộ scan.

## Bước 2 — Query OSV.dev (batch)

### 2a. Batch query lấy vuln IDs

```bash
curl -s -X POST https://api.osv.dev/v1/querybatch \
  -H "Content-Type: application/json" \
  -d '{
    "queries": [
      {"package": {"name": "Newtonsoft.Json", "ecosystem": "NuGet"}, "version": "12.0.1"},
      {"package": {"name": "lodash", "ecosystem": "npm"}, "version": "4.17.20"}
    ]
  }'
```

Response mỗi query trả về **chỉ ID** (không có severity/fixed-version):

```json
{"results": [{"vulns": [{"id": "GHSA-5crp-9r3c-p9vr"}]}, {"vulns": [{"id": "GHSA-29mw-wpgm-hmr9"}]}]}
```

Batch theo nhóm ~500 package/lần gọi (giới hạn thực tế cho payload hợp lý — với repo thường <500 dependency, 1 lần gọi là đủ).

### 2b. Lấy chi tiết từng vuln ID (dedup trước khi gọi)

```bash
curl -s https://api.osv.dev/v1/vulns/GHSA-5crp-9r3c-p9vr
```

Trích từ response:
- `aliases[]` — tìm entry dạng `CVE-YYYY-NNNNN` → dùng làm `cve_id` (nếu không có alias CVE nào, dùng chính OSV id làm `cve_id` fallback)
- `severity[].score` (CVSS vector, vd `CVSS:3.1/AV:N/...`) hoặc `database_specific.severity` (string `CRITICAL`/`HIGH`/...) — xem Bước 3 để map ra severity
- `affected[].ranges[].events[]` — tìm event có `"fixed": "<version>"` → dùng làm `fixed_version` (lấy version fixed nhỏ nhất lớn hơn version đang dùng, nếu có nhiều range)
- `summary` — 1 dòng mô tả, dùng cho `issue_summary`

### 2c. Graceful degradation (BẮT BUỘC)

Nếu `curl` không có sẵn, network timeout, DNS fail, hoặc response không phải JSON hợp lệ:
1. In `{msg_sca_unavailable}` vào report (1 dòng, không phải lỗi block scan).
2. Fallback: dùng static list trong [`../rules/generic/20-outdated-dependency.md`](../rules/generic/20-outdated-dependency.md) — rule đó vẫn chạy độc lập, không phụ thuộc `--sca`.
3. KHÔNG retry network call nhiều lần (1 lần thử là đủ, tránh treo scan vì mạng chậm).

## Bước 3 — Map CVSS → severity

| CVSS score | Severity |
|---|---|
| ≥ 9.0 | CRITICAL |
| ≥ 7.0 | HIGH |
| ≥ 4.0 | MEDIUM |
| < 4.0 | LOW |

Nếu OSV response có `database_specific.severity` dạng string sẵn (`CRITICAL`/`HIGH`/`MODERATE`/`LOW`) mà không có CVSS score số, dùng trực tiếp (map `MODERATE` → `MEDIUM`). Nếu không có severity nào cả (một số advisory cũ) → mặc định `MEDIUM`, note "severity không rõ, mặc định MEDIUM" trong `issue_summary`.

Severity cuối cùng vẫn bị cap bởi `severity_max: CRITICAL` của rule 22 (xem rule file) — nhưng vì cap đã là CRITICAL, mapping trên áp dụng nguyên vẹn.

## Bước 4 — Tạo finding

Với mỗi `(package, version)` có ≥1 vuln xác nhận từ OSV:

```
rule_id: VULNERABLE-DEPENDENCY
file: <đường dẫn manifest, vd api/pkg.csproj>
line: <dòng chứa PackageReference/dependency đó trong manifest>
severity: <map từ Bước 3>
issue_summary: "<package>@<installed_version> có <cve_id> (<summary ngắn>)"
fix_summary: "Nâng cấp lên <fixed_version>"
# Fields riêng cho JSON canonical (xem output-format.md):
cve_id, osv_id, package, ecosystem, installed_version, fixed_version, cvss_score
```

Nếu 1 package có nhiều vuln id → tạo nhiều finding riêng (đúng nguyên tắc "1 finding = 1 rule_id" đã áp dụng cho toàn skill, mỗi vuln là 1 finding dù cùng `rule_id: VULNERABLE-DEPENDENCY`).

**Ưu tiên rule 22 hơn rule 20:** nếu cùng 1 package/version vừa nằm trong static list của rule 20 VÀ được OSV live xác nhận, chỉ tạo finding rule 22 (bỏ finding rule 20 trùng) — rule 22 có data chính xác hơn (fixed_version thật từ OSV thay vì static list có thể lỗi thời).

## Bước 5 — Enrich JSON canonical

Thêm block top-level vào JSON summary cuối report (xem chi tiết field ở [`output-format.md`](output-format.md) mục 7):

```json
"dependencies_scanned": {
  "total": 47,
  "ecosystems": ["NuGet", "npm"],
  "vulnerable_count": 3,
  "scan_source": "osv.dev live"
}
```

`scan_source` = `"static list (offline)"` nếu Bước 2c fallback xảy ra (network unavailable).

## Cross-references

- [`../rules/generic/22-vulnerable-dependency.md`](../rules/generic/22-vulnerable-dependency.md) — rule chính dùng data từ file này
- [`../rules/generic/20-outdated-dependency.md`](../rules/generic/20-outdated-dependency.md) — fallback offline, cross-check tránh double-report
- [`../workflows/auto-fix.md`](../workflows/auto-fix.md) — khi kết hợp `--sca --auto-fix`, finding `VULNERABLE-DEPENDENCY` được auto-fix bằng cách bump version trong manifest + `restore`/`install` lại trước khi build verify
