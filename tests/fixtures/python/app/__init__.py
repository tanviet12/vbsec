from flask import Flask
from sqlalchemy import create_engine

app = Flask(__name__)
engine = create_engine("sqlite:///shop.db")

from app import users, orders, tasks, backup, cache, auth, fetch, profile, files  # noqa: E402,F401
