<?php
require __DIR__ . '/../vendor/autoload.php';

use App\Auth;
use App\InvoiceRepository;

$userId = Auth::userId();
$repo = new InvoiceRepository();
$rows = isset($_GET['sort'])
    ? $repo->listSorted($userId, (string) $_GET['sort'])
    : $repo->invoicesForUser($userId);

header('Content-Type: application/json');
echo json_encode($rows);
