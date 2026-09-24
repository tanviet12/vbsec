<?php
require 'config.php';

$cart = [];
if (isset($_COOKIE['cart'])) {
    $cart = unserialize(base64_decode($_COOKIE['cart']));
}

echo 'Items in cart: ' . count($cart);
