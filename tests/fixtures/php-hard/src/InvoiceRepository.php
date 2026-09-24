<?php
namespace App;

use PDO;

final class InvoiceRepository
{
    public function invoicesForUser(int $userId): array
    {
        $pdo = Db::pdo();
        $stmt = $pdo->prepare('SELECT display_name FROM users WHERE id = ?');
        $stmt->execute([$userId]);
        $name = $stmt->fetchColumn();

        $sql = "SELECT id, amount, created_at FROM invoices WHERE customer_name = '" . $name . "' ORDER BY created_at DESC";
        return $pdo->query($sql)->fetchAll(PDO::FETCH_ASSOC);
    }

    public function listSorted(int $userId, string $sort): array
    {
        $allowed = ['amount', 'created_at'];
        $column = in_array($sort, $allowed, true) ? $sort : 'created_at';
        $stmt = Db::pdo()->prepare("SELECT id, amount, created_at FROM invoices WHERE user_id = ? ORDER BY {$column} DESC");
        $stmt->execute([$userId]);
        return $stmt->fetchAll(PDO::FETCH_ASSOC);
    }
}
