---
id: COMMAND-INJECTION
severity_max: CRITICAL
applies_to: dotnet
overrides: generic/21-command-injection.md
---

# Command Injection — .NET 10 Specialization

> Override cho rule chung `rules/generic/21-command-injection.md`. Áp dụng khi `primary_language: dotnet`.

## Intent (.NET-specific)

.NET code thường gọi tool hệ thống qua `System.Diagnostics.Process`: ImageMagick/ffmpeg/git/tar/powershell/cmd. Rủi ro xuất hiện khi input từ request được ghép vào `Arguments`, gọi `cmd.exe /c`, `powershell -Command`, hoặc dùng `ProcessStartInfo` với shell parsing. Kể cả khi `UseShellExecute = false`, một chuỗi `Arguments` tự ghép vẫn có thể bị option injection hoặc command injection nếu chạy shell.

## Khi nào CRITICAL

- `Process.Start("cmd.exe", "/c ... " + userInput)` hoặc `Process.Start("powershell", "-Command ...")`.
- `ProcessStartInfo.Arguments = $"... {input} ..."` với input L1 và gọi tool nguy hiểm.
- `UseShellExecute = true` với file/args từ request.
- `FileName` lấy từ request (`Process.Start(toolFromUser, ...)`).
- Chạy compiler/script/interpreter (`bash`, `sh`, `cmd`, `powershell`, `python`, `node`) với command string từ request.
- Upload filename/path đưa vào command mà không whitelist/canonicalize.

## Khi nào HIGH (giảm cấp)

- `ArgumentList.Add(input)` với `UseShellExecute = false` thường safe khỏi shell injection, nhưng vẫn kiểm tra path traversal/option injection.
- Input đã whitelist chặt (`^[a-zA-Z0-9._-]+$`) và không chạy qua shell.
- Endpoint admin/internal-only.
- Command chạy trong sandbox/container user thấp quyền.

## Cách reasoning

1. Grep process sinks: `Process.Start`, `ProcessStartInfo`, `Arguments`, `ArgumentList`, `UseShellExecute`, `cmd.exe`, `powershell`.
2. Read full method and trace file/argument source:
   - route/query/body/form/upload filename → L1.
   - DB field user-controlled → L2.
   - config constant → L3/L4.
3. Determine execution mode:
   - shell (`cmd /c`, `powershell -Command`, `UseShellExecute=true`) → CRITICAL with L1.
   - direct executable + `ArgumentList` → usually safe command-wise.
   - direct executable + single `Arguments` string → inspect quoting/whitelist.
4. Verify mitigations: allowlist command names, fixed executable path, `ArgumentList`, canonicalized file path, timeout, low-privilege execution.

## Search patterns

```
# Direct Process.Start with shell
Process\.Start\s*\(\s*["'](cmd\.exe|cmd|powershell|pwsh|bash|sh)["']
Process\.Start\s*\([^,]+,\s*\$
Process\.Start\s*\([^,]+,\s*["'][^"']*["']\s*\+

# ProcessStartInfo unsafe args
new\s+ProcessStartInfo\s*\(
\.Arguments\s*=\s*\$
\.Arguments\s*=\s*["'][^"']*["']\s*\+
UseShellExecute\s*=\s*true
\.FileName\s*=\s*(Request\.|.*Query|.*Form|.*Body)

# Dangerous interpreters/tools
(cmd\.exe|powershell|pwsh|bash|sh|python|node|ffmpeg|convert|magick|git|tar|zip)
```

## Examples

### CRITICAL — flag

```csharp
// cmd.exe /c with query input
app.MapPost("/convert", (string input) =>
{
    Process.Start("cmd.exe", $"/c magick convert {input} output.png");
});
// input = "a.png & type C:\\Windows\\win.ini"
```

```csharp
// ProcessStartInfo.Arguments concat from form
var file = Request.Form["file"].ToString();
var psi = new ProcessStartInfo
{
    FileName = "ffmpeg",
    Arguments = "-i " + file + " /tmp/out.mp4",
    UseShellExecute = true
};
Process.Start(psi);
```

```csharp
// User-controlled executable
var tool = Request.Query["tool"].ToString();
var arg = Request.Query["arg"].ToString();
Process.Start(tool, arg);
```

```csharp
// PowerShell command string
var name = request.FileName;
Process.Start("powershell", $"-Command Compress-Archive {name} out.zip");
```

### NOT critical — không flag

```csharp
// Fixed executable + ArgumentList + shell disabled
var psi = new ProcessStartInfo
{
    FileName = "/usr/bin/magick",
    UseShellExecute = false,
    RedirectStandardOutput = true
};
psi.ArgumentList.Add("convert");
psi.ArgumentList.Add(inputPath);
psi.ArgumentList.Add(outputPath);
Process.Start(psi);
```

```csharp
// Whitelist file name before passing as argument
if (!Regex.IsMatch(fileName, @"^[a-zA-Z0-9._-]+$"))
    return Results.BadRequest();

psi.ArgumentList.Add(Path.Combine(uploadRoot, fileName));
```

```csharp
// Fixed command selection
var tools = new Dictionary<string, string>
{
    ["thumbnail"] = "/usr/bin/magick",
    ["probe"] = "/usr/bin/ffprobe"
};
if (!tools.TryGetValue(kind, out var executable)) return Results.BadRequest();
```

## Fix recommendation

1. Avoid shell (`cmd.exe /c`, `powershell -Command`, `bash -c`) for user-controlled workflows.
2. Use fixed executable path and `ProcessStartInfo.ArgumentList` instead of one `Arguments` string.
3. Keep `UseShellExecute = false`.
4. Whitelist command mode, file extension, and file name; canonicalize paths under an allowed root.
5. Set timeout, clean environment, low-privilege user/container when running external tools.
6. Treat upload filenames as untrusted; generate server-side names before calling processing tools.

## Cross-references

- `10-path-traversal`: command may be safe but file path may still escape allowed directory.
- `16-unrestricted-file-upload`: uploaded file processing commonly reaches `ffmpeg`/ImageMagick.
- `08-insecure-deserialization`: both can become server-side RCE.
