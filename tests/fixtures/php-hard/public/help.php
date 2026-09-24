<?php
$page = $_GET['page'] ?? 'pages/index';
if (strpos($page, 'pages/') !== 0) {
    $page = 'pages/index';
}
include __DIR__ . '/' . $page . '.php';
