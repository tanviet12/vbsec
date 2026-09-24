<?php
namespace App;

final class ProfileRepository
{
    public function updateDisplayName(int $userId, string $name): void
    {
        $stmt = Db::pdo()->prepare('UPDATE users SET display_name = ? WHERE id = ?');
        $stmt->execute([$name, $userId]);
    }
}
