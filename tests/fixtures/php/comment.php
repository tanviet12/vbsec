<?php
require 'config.php';

$author = $_POST['author'] ?? 'anonymous';
$message = $_POST['message'] ?? '';
?>
<div class="comment">
    <strong><?= htmlspecialchars($author) ?></strong>
    <p><?= $message ?></p>
</div>
