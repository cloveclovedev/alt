"""Config CRUD operations."""

import json
from typing import Any

from .connection import NeonHTTP


def get(db: NeonHTTP, key: str, default: Any = None) -> Any:
    """Get a config value by key. Returns the deserialized JSON value, or default if not found."""
    result = db.execute("SELECT value FROM config WHERE key = $1", [key])
    if not result.rows:
        return default
    value = result.rows[0][0]
    if isinstance(value, str):
        try:
            return json.loads(value)
        except json.JSONDecodeError:
            return value
    return value


def set(db: NeonHTTP, key: str, value: Any) -> None:
    """Upsert a config value. Value is auto-JSON-encoded."""
    db.execute(
        """INSERT INTO config (key, value) VALUES ($1, $2::jsonb)
           ON CONFLICT (key) DO UPDATE
             SET value = EXCLUDED.value, updated_at = now()""",
        [key, json.dumps(value)],
    )
