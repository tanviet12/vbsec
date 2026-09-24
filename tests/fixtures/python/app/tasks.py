import subprocess

from flask import request, jsonify

from app import app


@app.post("/tools/ping")
def ping():
    host = request.json.get("host", "")
    result = subprocess.run(f"ping -c 1 {host}", shell=True, capture_output=True, text=True)
    return jsonify({"output": result.stdout})
