from flask import request, jsonify
from sqlalchemy import text

from app import app, engine


@app.get("/users/search")
def search_users():
    name = request.args.get("name", "")
    with engine.connect() as conn:
        rows = conn.execute(text(f"SELECT id, name, email FROM users WHERE name LIKE '%{name}%'"))
        return jsonify([dict(r._mapping) for r in rows])
