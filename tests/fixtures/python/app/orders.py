from flask import request, jsonify, abort
from flask_login import current_user, login_required
from sqlalchemy import text

from app import app, engine


@app.get("/orders")
@login_required
def list_orders():
    status = request.args.get("status", "paid")
    with engine.connect() as conn:
        rows = conn.execute(
            text("SELECT id, total FROM orders WHERE user_id = :uid AND status = :status"),
            {"uid": current_user.id, "status": status},
        )
        return jsonify([dict(r._mapping) for r in rows])


@app.get("/orders/<int:order_id>")
@login_required
def get_order(order_id):
    with engine.connect() as conn:
        row = conn.execute(
            text("SELECT id, user_id, total, address FROM orders WHERE id = :id"),
            {"id": order_id},
        ).first()
    if row is None:
        abort(404)
    return jsonify(dict(row._mapping))
