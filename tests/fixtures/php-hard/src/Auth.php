<?php
namespace App;

final class Auth
{
    public static function userId(): int
    {
        session_start();
        if (empty($_SESSION['user_id'])) {
            http_response_code(401);
            exit;
        }
        return (int) $_SESSION['user_id'];
    }
}
