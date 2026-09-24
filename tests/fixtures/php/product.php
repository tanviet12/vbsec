<?php
require 'config.php';
require 'db.php';

$id = $_GET['id'] ?? 0;
$stmt = $pdo->prepare('SELECT id, name, price FROM products WHERE id = ?');
$stmt->execute([$id]);
$product = $stmt->fetch(PDO::FETCH_ASSOC);

header('Content-Type: application/json');
echo json_encode($product ?: []);
