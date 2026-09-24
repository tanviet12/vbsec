<?php
require 'config.php';

$name = $_GET['name'] ?? 'guest';
?>
<h1>Hello, <?= htmlspecialchars($name, ENT_QUOTES, 'UTF-8') ?></h1>
