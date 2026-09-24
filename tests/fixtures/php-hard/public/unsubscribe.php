<?php
require __DIR__ . '/../vendor/autoload.php';

use App\Db;

$id = (int) ($_GET['id'] ?? 0);
Db::pdo()->prepare('UPDATE subscribers SET active = 0 WHERE id = ?')->execute([$id]);
echo 'You have been unsubscribed.';
