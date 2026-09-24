<?php
require __DIR__ . '/../vendor/autoload.php';

use App\Auth;

$userId = Auth::userId();
$file = $_FILES['avatar'] ?? null;
if (!$file || $file['error'] !== UPLOAD_ERR_OK) {
    http_response_code(400);
    exit;
}

$ext = pathinfo($file['name'], PATHINFO_EXTENSION);
if ($ext === 'php' || !str_starts_with($file['type'], 'image/')) {
    http_response_code(415);
    exit;
}

$dest = __DIR__ . '/uploads/' . $userId . '-' . basename($file['name']);
move_uploaded_file($file['tmp_name'], $dest);
echo json_encode(['url' => '/uploads/' . basename($dest)]);
