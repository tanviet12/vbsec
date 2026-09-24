import os

from flask import request, send_file

from app import app

UPLOAD_DIR = "/srv/shop/uploads"


@app.get("/download")
def download():
    name = request.args.get("file", "")
    return send_file(os.path.join(UPLOAD_DIR, name))
