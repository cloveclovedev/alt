"""Config CRUD operations."""

import json

from .connection import NeonHTTP


def get(db: NeonHTTP, key: str, default=None):
    """Get a config value by key. Returns the deserialized JSON value, or default if not found."""
    result = db.execute("SELECT value FROM config WHERE key = $1", [key])
    if not result.rows:
        return default
    value = result.rows[0][0]
    return json.loads(value) if isinstance(value, str) else value
