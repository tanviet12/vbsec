---
id: INSECURE-DESERIALIZATION
severity_max: CRITICAL
applies_to: dotnet
overrides: generic/08-insecure-deserialization.md
---

# Insecure Deserialization — .NET 10 Specialization

> Override cho rule chung `rules/generic/08-insecure-deserialization.md`. Áp dụng khi `primary_language: dotnet`.

## Intent (.NET-specific)

`System.Text.Json` mặc định tương đối an toàn, nhưng .NET codebase vẫn hay có Newtonsoft.Json legacy, BinaryFormatter/LosFormatter/NetDataContractSerializer, hoặc custom polymorphic converter. Nguy hiểm nhất là deserialize untrusted data với type metadata (`TypeNameHandling`) hoặc formatter có khả năng materialize arbitrary type/gadget chain. Trong ASP.NET Core, source thường là request body, uploaded file, cookie, cache, queue message hoặc webhook.

## Khi nào CRITICAL

- `JsonConvert.DeserializeObject` với `TypeNameHandling.Auto/All/Objects/Arrays` trên dữ liệu L1/L2.
- `JsonSerializerSettings.TypeNameHandling != None` mà không có strict `SerializationBinder`.
- `BinaryFormatter.Deserialize`, `LosFormatter.Deserialize`, `NetDataContractSerializer.ReadObject`, `SoapFormatter.Deserialize`.
- `DataContractSerializer` / `XmlSerializer` deserialize type do user chọn hoặc XML untrusted kèm unsafe resolver.
- Custom `JsonConverter` / polymorphism map cho phép type discriminator từ request map tự do sang runtime type.
- Deserialize từ cookie/session/cache/queue mà attacker có thể ghi hoặc ký không được verify.

## Khi nào HIGH (giảm cấp)

- Deserialize internal trusted file/config build-time.
- `System.Text.Json.JsonSerializer.Deserialize<TKnownDto>` vào DTO cố định, không polymorphic arbitrary type.
- Newtonsoft có `TypeNameHandling.None` hoặc binder allowlist chặt.
- Payload có HMAC/signature verify trước khi deserialize unsafe format.

## Cách reasoning

1. Grep deserialization sink: `JsonConvert.DeserializeObject`, `TypeNameHandling`, `BinaryFormatter`, `LosFormatter`, `NetDataContractSerializer`, `ReadObject`.
2. Read callsite và settings.
3. Trace payload source:
   - `Request.Body`, uploaded file, header/cookie, SignalR/WebSocket, queue/webhook → L1/L2.
   - local trusted config/sealed resource → L3/L4.
4. Verify:
   - Target type generic is DTO? `DeserializeObject<MyDto>(json)` safe hơn `object`.
   - `TypeNameHandling` có `None` không?
   - Có binder allowlist (`ISerializationBinder`) không?
   - XML resolver disabled?
5. Flag CRITICAL khi untrusted bytes/string đi vào unsafe deserializer/config.

## Search patterns

```
# Newtonsoft.Json type name handling
TypeNameHandling\s*=\s*TypeNameHandling\.(All|Auto|Objects|Arrays)
JsonConvert\.DeserializeObject\s*<\s*object\s*>
JsonConvert\.DeserializeObject\s*\([^)]*JsonSerializerSettings

# Legacy unsafe formatters
BinaryFormatter\s*\(
\.Deserialize\s*\(
LosFormatter\s*\(
NetDataContractSerializer\s*\(
SoapFormatter\s*\(

# Data/XML deserialization
DataContractSerializer\s*\(
XmlSerializer\s*\(
\.ReadObject\s*\(

# ASP.NET request sources near deserialize
Request\.Body
ReadAsStringAsync\s*\(
Request\.Form\.Files
Request\.Cookies
```

## Examples

### CRITICAL — flag

```csharp
// Newtonsoft TypeNameHandling.All from request body
[HttpPost("/import")]
public async Task<IActionResult> Import()
{
    using var reader = new StreamReader(Request.Body);
    var json = await reader.ReadToEndAsync();
    var obj = JsonConvert.DeserializeObject<object>(json, new JsonSerializerSettings
    {
        TypeNameHandling = TypeNameHandling.All
    });
    return Ok(obj);
}
```

```csharp
// BinaryFormatter from uploaded file
using var stream = formFile.OpenReadStream();
var formatter = new BinaryFormatter();
var state = formatter.Deserialize(stream);
```

```csharp
// NetDataContractSerializer materializes CLR types
var serializer = new NetDataContractSerializer();
var obj = serializer.ReadObject(Request.Body);
```

```csharp
// Polymorphic discriminator from request maps directly to Type
var typeName = doc.RootElement.GetProperty("$type").GetString();
var type = Type.GetType(typeName!);
return JsonSerializer.Deserialize(json, type!);
```

### NOT critical — không flag

```csharp
// System.Text.Json into known DTO
var req = await JsonSerializer.DeserializeAsync<ImportRequest>(Request.Body);
```

```csharp
// Newtonsoft fixed DTO and no type names
var req = JsonConvert.DeserializeObject<ImportRequest>(json, new JsonSerializerSettings
{
    TypeNameHandling = TypeNameHandling.None
});
```

```csharp
// Newtonsoft with binder allowlist
var settings = new JsonSerializerSettings
{
    TypeNameHandling = TypeNameHandling.Objects,
    SerializationBinder = new KnownTypesBinder(new[] { typeof(SafeMessage) })
};
```

```csharp
// Signed payload before legacy load (acceptable with strong key)
if (!hmacValidator.IsValid(payload, signature)) return Unauthorized();
var obj = formatter.Deserialize(new MemoryStream(payload));
```

## Fix recommendation

1. Prefer `System.Text.Json` into known DTO/record types.
2. Do not enable Newtonsoft `TypeNameHandling` for untrusted payloads. If unavoidable, use strict `ISerializationBinder` allowlist.
3. Remove `BinaryFormatter`, `LosFormatter`, `NetDataContractSerializer`, `SoapFormatter` from request/cookie/upload paths.
4. For polymorphism, use explicit discriminator enum/map to known DTO types, never `Type.GetType(userInput)`.
5. Verify HMAC/signature before deserializing data from cache/queue/cookie if legacy format cannot be removed.

## Cross-references

- `21-command-injection`: unsafe deserialization often reaches RCE sinks.
- `01-hardcoded-secret`: HMAC/signing keys must not be hardcoded.
- `20-outdated-dependency`: old Newtonsoft/gadget-compatible libraries increase exploitability.
