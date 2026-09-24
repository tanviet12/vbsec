<?php
header('Content-Type: application/json; charset=utf-8');
echo json_encode(['message' => 'Hello, ' . ($_GET['name'] ?? 'guest')], JSON_HEX_TAG | JSON_HEX_AMP);
