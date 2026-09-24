<?php
require __DIR__ . '/../vendor/autoload.php';

use App\Db;

session_start();
$stmt = Db::pdo()->prepare('SELECT id, password FROM users WHERE email = ?');
$stmt->execute([$_POST['email'] ?? '']);
$user = $stmt->fetch();

if ($user && $user['password'] == md5($_POST['password'] ?? '')) {
    session_regenerate_id(true);
    $_SESSION['user_id'] = $user['id'];
    header('Location: /dashboard');
    exit;
}
http_response_code(401);
echo 'Invalid credentials';
