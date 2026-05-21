---
id: MASS-ASSIGNMENT
severity_max: CRITICAL
applies_to: dotnet
overrides: generic/07-mass-assignment.md
---

# Mass Assignment — .NET 10 / ASP.NET Core Specialization

> Override cho rule chung `rules/generic/07-mass-assignment.md`. Áp dụng khi `primary_language: dotnet`.

## Intent (.NET-specific)

ASP.NET Core model binding tự động map JSON/form/query vào action parameter. Nếu controller hoặc Minimal API nhận trực tiếp EF entity/domain model (`User`, `Account`, `Order`) rồi gọi `Update`, `Entry(...).State = Modified`, `SetValues`, hoặc save entity đó, attacker có thể gửi thêm field nhạy cảm như `IsAdmin`, `Role`, `Balance`, `TenantId`, `EmailConfirmed`, `Permissions` để leo quyền hoặc sửa dữ liệu ngoài ý muốn.

## Khi nào CRITICAL

- Action/endpoint nhận entity model trực tiếp từ `[FromBody]` hoặc parameter Minimal API: `User inputUser`.
- `DbContext.Update(input)`, `_db.Entry(input).State = EntityState.Modified`, `CurrentValues.SetValues(input)` với input bind từ request.
- `TryUpdateModelAsync(entity)` không có include-list field an toàn.
- DTO có field nhạy cảm (`IsAdmin`, `Role`, `TenantId`, `Balance`) dùng cho user-facing endpoint.
- AutoMapper map toàn bộ DTO/request vào entity mà không ignore sensitive fields.

## Khi nào HIGH (giảm cấp)

- Entity không có field nhạy cảm và endpoint chỉ update public profile fields.
- Có `[Bind("Name,Email")]` include-list rõ ràng hoặc DTO riêng chỉ chứa field cho phép.
- Endpoint admin-only rõ ràng và đổi field nhạy cảm là mục đích chính.
- Field nhạy cảm bị server override sau bind (`entity.Role = "User"`) trước save, nhưng vẫn đọc kỹ thứ tự assignment.

## Cách reasoning

1. Grep action/minimal API nhận model từ request: `[FromBody] User`, `MapPost(... User user ...)`, `TryUpdateModelAsync`, `SetValues`.
2. Read entity class để xác định field nhạy cảm: role, admin, tenant, owner, balance, permissions, status, verified.
3. Trace source:
   - MVC action parameter / Minimal API parameter / `Request.ReadFromJsonAsync<T>()` → L1.
   - DTO internal từ service khác → L3/L4 tùy context.
4. Verify mitigation:
   - DTO/record input riêng không chứa field nhạy cảm.
   - Manual mapping allowlist.
   - `[Bind]` include-list, not exclude-list.
   - AutoMapper profile ignore sensitive members.
5. Flag khi L1 object có thể set field ngoài allowlist trước khi save.

## Search patterns

```
# MVC / Minimal API binds entity-ish names
\[FromBody\]\s*(User|Account|Order|Customer|Member|Profile)\b
Map(Post|Put|Patch)\s*\([^)]*,\s*(async\s*)?\([^)]*\b(User|Account|Order|Customer)\s+\w+
ReadFromJsonAsync\s*<\s*(User|Account|Order|Customer)

# EF update whole object
\.Update\s*\(\s*\w+\s*\)
\.Entry\s*\([^)]+\)\.State\s*=\s*EntityState\.Modified
CurrentValues\.SetValues\s*\(
TryUpdateModelAsync\s*\(

# AutoMapper / mapper direct request to entity
\.Map\s*<\s*(User|Account|Order|Customer)\s*>\s*\(
mapper\.Map\s*\([^,]+,\s*\w+\s*\)

# Sensitive fields
\b(IsAdmin|IsSuperAdmin|Role|Roles|Permissions|TenantId|OwnerId|Balance|CreditLimit|EmailConfirmed|Status)\b
```

## Examples

### CRITICAL — flag

```csharp
// Minimal API bind trực tiếp entity
app.MapPut("/users/{id}", async (int id, User inputUser, AppDbContext db) =>
{
    inputUser.Id = id;
    db.Users.Update(inputUser);
    await db.SaveChangesAsync();
});
// Attacker body: {"name":"x","isAdmin":true,"role":"Admin"}
```

```csharp
// MVC + EntityState.Modified
[HttpPut("{id}")]
public async Task<IActionResult> Put(int id, [FromBody] User user)
{
    user.Id = id;
    _db.Entry(user).State = EntityState.Modified;
    await _db.SaveChangesAsync();
    return Ok();
}
```

```csharp
// SetValues copies all matching properties
var user = await _db.Users.FindAsync(id);
var input = await Request.ReadFromJsonAsync<User>();
_db.Entry(user).CurrentValues.SetValues(input);
await _db.SaveChangesAsync();
```

```csharp
// DTO still unsafe because it exposes sensitive fields
public record UpdateUserRequest(string Name, string Role, bool IsAdmin);

[HttpPatch("{id}")]
public async Task<IActionResult> Patch(int id, UpdateUserRequest req)
{
    _mapper.Map(req, await _db.Users.FindAsync(id));
    await _db.SaveChangesAsync();
    return Ok();
}
```

### NOT critical — không flag

```csharp
// DTO only has allowed fields
public record UpdateProfileRequest(string DisplayName, string Bio);

app.MapPut("/me", async (UpdateProfileRequest req, ClaimsPrincipal user, AppDbContext db) =>
{
    var me = await db.Users.SingleAsync(u => u.Id == user.UserId());
    me.DisplayName = req.DisplayName;
    me.Bio = req.Bio;
    await db.SaveChangesAsync();
});
```

```csharp
// Bind include-list
public async Task<IActionResult> Edit(
    [Bind("DisplayName,Bio,TimeZone")] UserProfile input)
{
    ...
}
```

```csharp
// Admin endpoint explicitly changes role and checks authorization
[Authorize(Roles = "Admin")]
[HttpPatch("/admin/users/{id}/role")]
public async Task<IActionResult> ChangeRole(int id, ChangeRoleRequest req)
{
    var user = await _db.Users.FindAsync(id);
    user.Role = req.Role;
    await _db.SaveChangesAsync();
    return Ok();
}
```

## Fix recommendation

1. Không nhận EF entity/domain model trực tiếp trong public action. Tạo DTO/record riêng cho từng use case.
2. Manual map field được phép; tránh `Update(entity)` với object bind từ request.
3. Dùng `[Bind("Allowed,Fields")]` nếu bắt buộc model binding MVC, nhưng DTO vẫn rõ hơn.
4. AutoMapper profile phải `Ignore()` field nhạy cảm (`Role`, `IsAdmin`, `TenantId`, `Balance`).
5. Field ownership/tenant/role phải lấy từ auth context hoặc server-side policy, không từ request body.

## Cross-references

- `12-broken-access-control`: mass assignment thường biến thành privilege escalation.
- `04-idor`: attacker đổi `OwnerId`/`TenantId` để chạm dữ liệu người khác.
- `02-sql-injection`: cùng nguồn L1 từ ASP.NET Core route/body binding.
