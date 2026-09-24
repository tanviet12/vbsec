<?php
require 'config.php';

$host = $_GET['host'] ?? '127.0.0.1';
$output = shell_exec('ping -c 1 ' . $host);

echo '<pre>' . htmlspecialchars($output ?? '') . '</pre>';
