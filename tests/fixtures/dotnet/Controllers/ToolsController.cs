using System.Diagnostics;
using Microsoft.AspNetCore.Mvc;

namespace Shop.Controllers;

[ApiController]
[Route("tools")]
public class ToolsController : ControllerBase
{
    [HttpPost("ping")]
    public IActionResult Ping([FromForm] string host)
    {
        var psi = new ProcessStartInfo("cmd.exe", "/c ping -n 1 " + host)
        {
            RedirectStandardOutput = true,
            UseShellExecute = false
        };
        using var proc = Process.Start(psi)!;
        var output = proc.StandardOutput.ReadToEnd();
        proc.WaitForExit();
        return Ok(new { output });
    }
}
