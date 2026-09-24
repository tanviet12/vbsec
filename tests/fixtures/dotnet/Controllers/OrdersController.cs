using System.Security.Claims;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using Shop.Data;

namespace Shop.Controllers;

[ApiController]
[Authorize]
[Route("orders")]
public class OrdersController : ControllerBase
{
    private readonly ShopDbContext _db;

    public OrdersController(ShopDbContext db) => _db = db;

    [HttpGet]
    public async Task<IActionResult> List([FromQuery] decimal minTotal)
    {
        var userId = int.Parse(User.FindFirstValue(ClaimTypes.NameIdentifier)!);
        var orders = await _db.Orders
            .FromSqlInterpolated($"SELECT * FROM Orders WHERE UserId = {userId} AND Total >= {minTotal}")
            .ToListAsync();
        return Ok(orders);
    }
}
