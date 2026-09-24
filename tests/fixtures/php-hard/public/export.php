<?php
require __DIR__ . '/../vendor/autoload.php';

use App\Auth;

$userId = Auth::userId();
$format = $_GET['format'] ?? 'pdf';
$source = sys_get_temp_dir() . '/statement-' . $userId . '.html';
$target = sys_get_temp_dir() . '/statement-' . $userId . '.' . preg_replace('/[^a-z]/', '', $format);

exec('pandoc ' . escapeshellarg($source) . ' --to=' . $format . ' -o ' . escapeshellarg($target), $out, $code);

if ($code !== 0) {
    http_response_code(500);
    exit;
}
readfile($target);
