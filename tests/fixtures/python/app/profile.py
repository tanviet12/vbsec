import requests
import yaml
from flask import request, jsonify
from flask_login import current_user, login_required

from app import app

WEATHER_API = "https://api.weather.example.com/v1/current"


@app.post("/profile/preferences")
@login_required
def update_preferences():
    prefs = yaml.safe_load(request.data)
    theme = prefs.get("theme", "light") if isinstance(prefs, dict) else "light"
    return jsonify({"user": current_user.id, "theme": theme})


@app.get("/profile/weather")
@login_required
def weather():
    resp = requests.get(WEATHER_API, params={"city": current_user.city}, timeout=5)
    return jsonify(resp.json())
