---
id: VULNERABLE-DEPENDENCY
severity_max: CRITICAL
applies_to: all
---

# Vulnerable Dependency (Live CVE — OSV.dev)

## Intent

Rule này khác với `20-outdated-dependency`: đây là finding **đã xác nhận** qua tra cứu live tới [OSV.dev](https://osv.dev), có `cve_id` cụ thể, `fixed_version` chính xác, và điểm CVSS — không phải suy đoán từ static list offline. Chỉ chạy khi user truyền flag `--sca` (xem [`../../references/dependency-scan.md`](../../references/dependency-scan.md) cho toàn bộ cơ chế parse manifest + gọi API).

Vibe coder pin version dependency từ tutorial/template cũ, không track CVE. Với data live từ OSV, vbsec biết chính xác: package nào, version nào, CVE nào, và version nào cần nâng cấp lên — đủ để CI/CD chặn merge một cách có cơ sở, không phải "nghi ngờ".

## Khi nào CRITICAL / HIGH / MEDIUM

| Condition | Severity |
|---|---|
| OSV trả CVSS ≥ 9.0, hoặc `database_specific.severity: CRITICAL` | CRITICAL |
| OSV trả CVSS 7.0–8.9, hoặc `severity: HIGH` | HIGH |
| OSV trả CVSS 4.0–6.9, hoặc `severity: MODERATE`/`MEDIUM` | MEDIUM |
| OSV trả CVSS < 4.0, hoặc `severity: LOW` | LOW |
| Không có severity nào trong response OSV | MEDIUM (mặc định, ghi rõ lý do trong `issue_summary`) |

## Reasoning

1. Đọc [`dependency-scan.md`](../../references/dependency-scan.md) để biết cách parse manifest cho ecosystem tương ứng (`NuGet`, `Go`, `npm`, `Packagist`, `PyPI`).
2. Gọi `POST /v1/querybatch` lấy vuln ID, rồi `GET /v1/vulns/{id}` lấy chi tiết (severity, fixed version, CVE alias).
3. Map CVSS/severity string sang severity vbsec (bảng trên).
4. Chỉ flag khi OSV **thực sự trả về ≥1 vuln** cho `(package, installed_version)` — không suy đoán, không dùng static list ở đây (đó là việc của rule 20).
5. Nếu network không khả dụng, rule này KHÔNG chạy (không có gì để flag) — rule 20 vẫn chạy độc lập làm fallback offline.
6. Nếu cùng package/version match cả rule này và rule 20, chỉ giữ finding của rule này (data chính xác hơn), bỏ finding trùng của rule 20.

## Search patterns (illustrative — KHÔNG phải lệnh chạy literal)

```bash
# Ví dụ minh họa 1 batch query — chi tiết đầy đủ ở dependency-scan.md
curl -s -X POST https://api.osv.dev/v1/querybatch -H "Content-Type: application/json" -d '{
  "queries": [{"package": {"name": "<pkg>", "ecosystem": "<NuGet|Go|npm|Packagist|PyPI>"}, "version": "<version>"}]
}'
```

## Examples

### CRITICAL — flag

```json
// composer.lock — guzzlehttp/guzzle 7.4.1, OSV xác nhận GHSA (CVE-2022-31091, CVSS 7.5 -> nhưng ví dụ dưới giả định 1 CVE RCE khác CVSS 9.8)
{
  "name": "guzzlehttp/guzzle",
  "version": "7.4.1"
}
```
→ OSV trả `severity: CRITICAL`, `fixed: "7.4.5"`, `aliases: ["CVE-2022-31091"]` → finding CRITICAL, fix = nâng lên `7.4.5`.

### NOT critical — không flag

```json
{
  "name": "guzzlehttp/guzzle",
  "version": "7.4.5"
}
```
→ OSV không trả vuln nào cho version này → không flag (đưa vào PASSED list nếu package này là dependency chính được scan).

## Fix recommendation

1. Nâng version package lên đúng `fixed_version` OSV trả về (không phải "latest" — có thể latest đã có breaking change không cần thiết):
   ```bash
   # NuGet
   dotnet add package <PackageName> --version <fixed_version>
   # npm
   npm install <package>@<fixed_version>
   # Go
   go get <module>@<fixed_version> && go mod tidy
   # Composer
   composer require <vendor/package>:<fixed_version>
   # pip
   pip install <package>==<fixed_version>
   ```
2. Sau khi bump version, chạy lại restore/install (`dotnet restore`, `npm install`, `go build ./...`, `composer update`, `pip install -r requirements.txt`) để lockfile khớp.
3. Nếu dùng `--auto-fix` cùng lúc, xem [`../../workflows/auto-fix.md`](../../workflows/auto-fix.md) mục "Special case — dependency-fix patches" — vbsec tự bump version + restore + build verify.
4. Nếu không thể upgrade ngay (breaking change lớn), note rõ risk acceptance trong PR + theo dõi lại ở lần scan sau.

## Cross-references

- `20-outdated-dependency`: fallback offline/static list khi không có `--sca` hoặc network không khả dụng. Không double-report cùng 1 package/version ở cả 2 rule.
- `01-hardcoded-secret`: một số CVE liên quan tới leak secret qua dependency (vd package độc hại exfiltrate env vars).
- OWASP: [A06:2021 – Vulnerable and Outdated Components](https://owasp.org/Top10/A06_2021-Vulnerable_and_Outdated_Components/)
- CWE: CWE-1104 (Use of Unmaintained Third Party Components)
