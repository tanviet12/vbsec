<?php
require __DIR__ . '/../vendor/autoload.php';

use App\Auth;
use App\ProfileRepository;

$userId = Auth::userId();
$name = trim($_POST['display_name'] ?? '');
if ($name === '' || mb_strlen($name) > 100) {
    http_response_code(422);
    exit;
}
(new ProfileRepository())->updateDisplayName($userId, $name);
header('Location: /profile');
