# .NET 10 Specialization Rules

Specialization này áp dụng cho `primary_language: dotnet`, bao phủ C# / ASP.NET Core / Minimal API / EF Core / Newtonsoft.Json / `System.Diagnostics.Process`.

## Rules

| Rule | File | Focus |
|---|---|---|
| SQL-INJECTION | `02-sql-injection.md` | EF Core raw SQL (`FromSqlRaw`, `ExecuteSqlRaw`), ADO.NET `SqlCommand` |
| MASS-ASSIGNMENT | `07-mass-assignment.md` | ASP.NET Core model binding trực tiếp vào entity/domain model |
| INSECURE-DESERIALIZATION | `08-insecure-deserialization.md` | Newtonsoft `TypeNameHandling`, BinaryFormatter/NetDataContractSerializer/LosFormatter legacy |
| COMMAND-INJECTION | `21-command-injection.md` | `Process.Start`, `ProcessStartInfo`, shell invocation, unsafe argument construction |

## Detection notes

`dotnet` nên được detect từ `.cs`, `.csproj`, `.sln`, `global.json`, `appsettings.json`. Khi repo có frontend TypeScript + backend ASP.NET Core, load cả `typescript` và `dotnet` overlay nếu cả hai vượt ngưỡng.

## Reasoning

Các rule này không flag bằng pattern thuần. Agent vẫn phải trace L1-L4:

1. L1: route/query/body/header/form/file upload được ASP.NET Core bind tự động.
2. Sink: raw SQL, entity update, unsafe deserializer, process execution.
3. Sanitization/mitigation: parameterized EF/ADO.NET, DTO allowlist, safe serializer config, list-form arguments / whitelist.
4. Chỉ report khi input untrusted đi tới sink mà không có kiểm soát phù hợp.
