import requests
from flask import request, jsonify

from app import app


@app.get("/preview")
def preview():
    url = request.args["url"]
    resp = requests.get(url, timeout=5)
    return jsonify({"status": resp.status_code, "body": resp.text[:500]})
