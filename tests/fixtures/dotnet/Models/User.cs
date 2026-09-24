namespace Shop.Models;

public class User
{
    public int Id { get; set; }
    public string Email { get; set; } = "";
    public string DisplayName { get; set; } = "";
    public bool IsAdmin { get; set; }
    public decimal Balance { get; set; }
}
