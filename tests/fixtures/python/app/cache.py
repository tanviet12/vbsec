import base64
import pickle

from flask import request, jsonify

from app import app


@app.post("/cart/restore")
def restore_cart():
    blob = request.cookies.get("cart", "")
    cart = pickle.loads(base64.b64decode(blob))
    return jsonify({"items": len(cart)})
