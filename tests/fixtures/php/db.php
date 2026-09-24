<?php
$conn = mysqli_connect('localhost', 'shop', getenv('DB_PASSWORD'), 'shop');
$pdo = new PDO('mysql:host=localhost;dbname=shop', 'shop', getenv('DB_PASSWORD'));
