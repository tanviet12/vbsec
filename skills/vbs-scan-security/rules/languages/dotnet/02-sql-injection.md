---
id: SQL-INJECTION
severity_max: CRITICAL
applies_to: dotnet
overrides: generic/02-sql-injection.md
---

# SQL Injection — .NET 10 / EF Core Specialization

> Override cho rule chung `rules/generic/02-sql-injection.md`. Áp dụng khi `primary_language: dotnet`.

## Intent (.NET-specific)

.NET 10 / ASP.NET Core app thường dùng EF Core, Dapper hoặc ADO.NET. EF Core query LINQ typed thường safe, nhưng các API raw SQL như `FromSqlRaw`, `ExecuteSqlRaw`, `SqlQueryRaw`, Dapper `Query($"...")`, hoặc `SqlCommand.CommandText` ghép string vẫn tạo SQL Injection nếu nhận dữ liệu từ route/query/body. Vibe code hay nhầm interpolated string trong `FromSqlRaw($"...{input}...")` là parameterized, nhưng thực tế string đã được build trước khi EF nhận query.

## Khi nào CRITICAL

- `FromSqlRaw`, `ExecuteSqlRaw`, `SqlQueryRaw` nhận interpolated string, concat, `string.Format`, hoặc `$"...{input}..."`.
- ADO.NET `new SqlCommand("... " + input, conn)` hoặc `cmd.CommandText = $"...{input}..."` mà không dùng `SqlParameter`.
- Dapper `Query`, `Execute`, `QueryAsync` nhận SQL string build từ L1 mà không có anonymous object params.
- Input từ ASP.NET Core route/query/body/header/form đi vào raw SQL public endpoint.
- Dynamic `ORDER BY`, table name, column name nhận trực tiếp từ request mà không whitelist.

## Khi nào HIGH (giảm cấp)

- Endpoint chỉ admin gọi được nhưng vẫn dùng raw SQL unsafe.
- Input đã ép kiểu mạnh trước khi concat (`int.TryParse`) nhưng vẫn nên parameterize.
- Dynamic identifier đã qua whitelist rõ ràng thì không flag; nếu whitelist thiếu chặt, downgrade thay vì CRITICAL.
- Query chỉ chạy trong migration/seed/test code, không reachable từ request.

## Cách reasoning

1. Grep raw SQL sink: `FromSqlRaw`, `ExecuteSqlRaw`, `SqlQueryRaw`, `SqlCommand`, Dapper `Query/Execute`.
2. Read controller/minimal API/service chứa sink.
3. Trace biến trong SQL string:
   - ASP.NET Core action parameter, `HttpContext.Request.Query`, `[FromBody]`, `[FromForm]`, route `{id}` → L1.
   - DB record dùng lại trong SQL → L2.
   - config/env/internal enum → L3.
   - constant string → L4.
4. Verify mitigation:
   - EF Core: `FromSql` / `FromSqlInterpolated` với interpolation parameterized.
   - ADO.NET: `cmd.Parameters.Add(...)` / `AddWithValue(...)`.
   - Dapper: `connection.Query(sql, new { id })`.
   - Identifier dynamic: allowlist enum/map.
5. Chỉ flag khi L1/L2 đi vào SQL sink mà không parameterize/whitelist.

## Search patterns

```
# EF Core raw SQL
\.FromSqlRaw\s*\(\s*\$
\.FromSqlRaw\s*\(\s*["'][^"']*["']\s*\+
\.ExecuteSqlRaw(Async)?\s*\(\s*\$
\.SqlQueryRaw\s*<[^>]+>\s*\(\s*\$

# ADO.NET
new\s+SqlCommand\s*\(\s*\$
new\s+SqlCommand\s*\(\s*["'][^"']*["']\s*\+
\.CommandText\s*=\s*\$
\.CommandText\s*\+=

# Dapper
\.(Query|QueryAsync|Execute|ExecuteAsync)\s*\(\s*\$
\.(Query|QueryAsync|Execute|ExecuteAsync)\s*\(\s*["'][^"']*["']\s*\+

# String helpers near SQL
string\.Format\s*\(\s*["'][^"']*(SELECT|INSERT|UPDATE|DELETE)
\$["'][^"']*(SELECT|INSERT|UPDATE|DELETE)[^"']*\{
```

## Examples

### CRITICAL — flag

```csharp
// ASP.NET Core Minimal API + EF Core FromSqlRaw
app.MapGet("/users", async (string name, AppDbContext db) =>
{
    return await db.Users
        .FromSqlRaw($"SELECT * FROM Users WHERE Name = '{name}'")
        .ToListAsync();
});
```

```csharp
// ExecuteSqlRawAsync với body input
[HttpDelete]
public Task Delete([FromBody] DeleteUserRequest req)
{
    return _db.Database.ExecuteSqlRawAsync(
        $"DELETE FROM Users WHERE Email = '{req.Email}'");
}
```

```csharp
// ADO.NET CommandText concat
var id = HttpContext.Request.Query["id"];
using var cmd = new SqlCommand("SELECT * FROM Orders WHERE Id = " + id, conn);
using var reader = cmd.ExecuteReader();
```

```csharp
// Dapper without params
var sort = Request.Query["sort"].ToString();
var rows = await connection.QueryAsync<Order>(
    $"SELECT * FROM Orders ORDER BY {sort}");
```

### NOT critical — không flag

```csharp
// EF Core FromSql parameterizes interpolated values
var users = await db.Users
    .FromSql($"SELECT * FROM Users WHERE Name = {name}")
    .ToListAsync();
```

```csharp
// Explicit interpolated API
var users = await db.Users
    .FromSqlInterpolated($"SELECT * FROM Users WHERE Email = {email}")
    .ToListAsync();
```

```csharp
// ADO.NET parameter
using var cmd = new SqlCommand("SELECT * FROM Orders WHERE Id = @id", conn);
cmd.Parameters.Add("@id", SqlDbType.Int).Value = id;
```

```csharp
// Dapper parameter object
var rows = await connection.QueryAsync<Order>(
    "SELECT * FROM Orders WHERE CustomerId = @customerId",
    new { customerId });
```

```csharp
// Dynamic ORDER BY with allowlist
var allowed = new Dictionary<string, string>
{
    ["created"] = "CreatedAt",
    ["name"] = "Name"
};
if (!allowed.TryGetValue(sort, out var col)) col = "CreatedAt";
var sql = $"SELECT * FROM Users ORDER BY {col}";
```

## Fix recommendation

1. EF Core: dùng `FromSql` hoặc `FromSqlInterpolated` cho value parameters; tránh `FromSqlRaw` với user input.
2. `ExecuteSqlRaw` chỉ dùng với placeholders + params, hoặc chuyển sang `ExecuteSql`/`ExecuteSqlInterpolated`.
3. ADO.NET: dùng `SqlParameter`, không concat vào `CommandText`.
4. Dapper: truyền anonymous object params, không build SQL từ request.
5. Với table/column/order dynamic: map từ allowlist hardcoded, không nhận string trực tiếp từ request.

## Cross-references

- `07-mass-assignment`: ASP.NET Core model binding cũng là đường L1 phổ biến.
- `17-verbose-error-debug-mode`: SQL error/stack trace leak giúp attacker refine payload.
- `04-idor`: SQLi cộng thiếu ownership check có thể dump dữ liệu tenant khác.
