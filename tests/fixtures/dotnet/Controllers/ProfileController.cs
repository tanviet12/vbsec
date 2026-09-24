using System.Security.Claims;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Shop.Data;
using Shop.Models;

namespace Shop.Controllers;

[ApiController]
[Authorize]
[Route("profile")]
public class ProfileController : ControllerBase
{
    private readonly ShopDbContext _db;

    public ProfileController(ShopDbContext db) => _db = db;

    [HttpPut]
    public async Task<IActionResult> Update([FromBody] User input)
    {
        input.Id = int.Parse(User.FindFirstValue(ClaimTypes.NameIdentifier)!);
        _db.Users.Update(input);
        await _db.SaveChangesAsync();
        return NoContent();
    }
}
