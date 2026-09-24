import subprocess
from datetime import date

from flask import jsonify, abort
from flask_login import current_user, login_required

from app import app

BACKUP_DIR = "/var/backups/shop"


@app.post("/admin/backup")
@login_required
def backup():
    if not current_user.is_admin:
        abort(403)
    archive = f"{BACKUP_DIR}/shop-{date.today().isoformat()}.tar.gz"
    subprocess.run(["tar", "-czf", archive, "/srv/shop/data"], check=True)
    return jsonify({"archive": archive})
