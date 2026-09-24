using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;
using Shop.Data;

namespace Shop.Controllers;

[ApiController]
[Route("users")]
public class UsersController : ControllerBase
{
    private readonly ShopDbContext _db;

    public UsersController(ShopDbContext db) => _db = db;

    [HttpGet("search")]
    public async Task<IActionResult> Search([FromQuery] string name)
    {
        var users = await _db.Users
            .FromSqlRaw($"SELECT * FROM Users WHERE DisplayName LIKE '%{name}%'")
            .ToListAsync();
        return Ok(users);
    }
}
