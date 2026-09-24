<?php
require 'config.php';
require 'db.php';

$q = $_GET['q'] ?? '';
$result = mysqli_query($conn, "SELECT id, name, price FROM products WHERE name LIKE '%" . $q . "%'");
$rows = mysqli_fetch_all($result, MYSQLI_ASSOC);

header('Content-Type: application/json');
echo json_encode($rows);
