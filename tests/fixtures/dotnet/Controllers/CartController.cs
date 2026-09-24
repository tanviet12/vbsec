using Microsoft.AspNetCore.Mvc;
using Newtonsoft.Json;

namespace Shop.Controllers;

[ApiController]
[Route("cart")]
public class CartController : ControllerBase
{
    private static readonly JsonSerializerSettings Settings = new()
    {
        TypeNameHandling = TypeNameHandling.All
    };

    [HttpPost("restore")]
    public IActionResult Restore()
    {
        var raw = Request.Cookies["cart"] ?? "[]";
        var cart = JsonConvert.DeserializeObject<List<object>>(raw, Settings);
        return Ok(new { items = cart?.Count ?? 0 });
    }
}
