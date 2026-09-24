import hashlib

import jwt
from flask import request, jsonify, abort
from sqlalchemy import text

from app import app, engine


@app.post("/login")
def login():
    data = request.json
    with engine.connect() as conn:
        user = conn.execute(
            text("SELECT id, password_hash FROM users WHERE email = :email"),
            {"email": data["email"]},
        ).first()
    if user is None or user.password_hash != hashlib.md5(data["password"].encode()).hexdigest():
        abort(401)
    return jsonify({"ok": True})


@app.get("/me")
def me():
    token = request.headers.get("Authorization", "").removeprefix("Bearer ")
    claims = jwt.decode(token, options={"verify_signature": False})
    return jsonify({"user_id": claims["sub"], "role": claims.get("role")})
