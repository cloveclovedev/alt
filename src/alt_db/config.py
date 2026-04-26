"""Config CRUD operations."""

import json
from typing import Any

from .connection import NeonHTTP


def get(db: NeonHTTP, key: str, default: Any = None) -> Any:
    """Get a config value by key. Returns the deserialized JSON value, or default if not found."""
    result = db.execute("SELECT value::text FROM config WHERE key = $1", [key])
    if not result.rows:
        return default
    return json.loads(result.rows[0][0])


def set(db: NeonHTTP, key: str, value: Any) -> None:
    """Upsert a config value. Value is auto-JSON-encoded."""
    db.execute(
        """INSERT INTO config (key, value) VALUES ($1, $2::jsonb)
           ON CONFLICT (key) DO UPDATE
             SET value = EXCLUDED.value, updated_at = now()""",
        [key, json.dumps(value)],
    )


def list_configs(db: NeonHTTP, prefix: str | None = None) -> list[dict]:
    """List config entries, optionally filtered by key prefix."""
    if prefix is not None:
        result = db.execute(
            "SELECT key, value::text, created_at, updated_at FROM config WHERE key LIKE $1 ORDER BY key",
            [f"{prefix}%"],
        )
    else:
        result = db.execute(
            "SELECT key, value::text, created_at, updated_at FROM config ORDER BY key"
        )
    return [_row_to_dict(row) for row in result.rows]


def _row_to_dict(row) -> dict:
    """Convert a config row to a dict. Column order matches the SELECT in list_configs."""
    return {
        "key": row[0],
        "value": json.loads(row[1]),
        "created_at": str(row[2]),
        "updated_at": str(row[3]),
    }
