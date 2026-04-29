"""Config CRUD operations."""

import json
import yaml
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


def delete(db: NeonHTTP, key: str) -> bool:
    """Delete a config entry. Returns True if deleted."""
    result = db.execute("DELETE FROM config WHERE key = $1", [key])
    return result.row_count > 0


def set_meta(db: NeonHTTP, key: str, metadata: dict) -> None:
    """Upsert metadata for a config key. Creates the row with value=null if absent.
    Does not modify value of existing rows."""
    db.execute(
        """INSERT INTO config (key, value, metadata) VALUES ($1, 'null'::jsonb, $2::jsonb)
           ON CONFLICT (key) DO UPDATE
             SET metadata = EXCLUDED.metadata, updated_at = now()""",
        [key, json.dumps(metadata)],
    )


def list_with_meta(db: NeonHTTP, prefix: str | None = None) -> list[dict]:
    """List config rows including metadata. Same shape as list_configs but
    each dict also has a 'metadata' field."""
    if prefix is not None:
        result = db.execute(
            "SELECT key, value::text, metadata::text, created_at, updated_at FROM config WHERE key LIKE $1 ORDER BY key",
            [f"{prefix}%"],
        )
    else:
        result = db.execute(
            "SELECT key, value::text, metadata::text, created_at, updated_at FROM config ORDER BY key"
        )
    return [_row_to_dict_with_meta(row) for row in result.rows]


def _row_to_dict_with_meta(row) -> dict:
    return {
        "key": row[0],
        "value": json.loads(row[1]),
        "metadata": json.loads(row[2]),
        "created_at": str(row[3]),
        "updated_at": str(row[4]),
    }


def load_yaml_defaults(path: str) -> dict[str, dict]:
    """Load and validate the YAML param catalog. Returns {key: meta_dict}."""
    with open(path) as f:
        raw = yaml.safe_load(f) or {}
    params = raw.get("params", {})
    if not isinstance(params, dict):
        raise ValueError(f"{path}: params must be a mapping (got {type(params).__name__})")
    out: dict[str, dict] = {}
    for key, meta in params.items():
        if not isinstance(meta, dict):
            raise ValueError(f"{path}: {key}: param entry must be a mapping")
        if "type" not in meta:
            raise ValueError(f"{path}: {key}: missing required field 'type'")
        if meta["type"] not in ("string", "number", "boolean", "array", "object"):
            raise ValueError(f"{path}: {key}: invalid type '{meta['type']}'")
        out[key] = meta
    return out


def seed(db: NeonHTTP, yaml_path: str, force: bool = False) -> dict[str, int]:
    """Idempotently apply a YAML param catalog to the config table.

    - Missing keys are inserted with value = meta.default (or NULL) and the
      full meta as metadata.
    - Existing keys are skipped unless force=True; with force, only the
      metadata is overwritten (the value is preserved).
    - Keys present in DB but absent from YAML are never touched.

    Returns a counts dict: {"inserted": int, "updated": int, "skipped": int}.
    """
    catalog = load_yaml_defaults(yaml_path)
    counts = {"inserted": 0, "updated": 0, "skipped": 0}
    for key, meta in catalog.items():
        existing = db.execute(
            "SELECT 1 FROM config WHERE key = $1",
            [key],
        )
        if existing.row_count == 0:
            default_value = meta.get("default", None)
            db.execute(
                """INSERT INTO config (key, value, metadata)
                   VALUES ($1, $2::jsonb, $3::jsonb)""",
                [key, json.dumps(default_value), json.dumps(meta)],
            )
            counts["inserted"] += 1
        elif force:
            db.execute(
                """UPDATE config SET metadata = $2::jsonb, updated_at = now()
                   WHERE key = $1""",
                [key, json.dumps(meta)],
            )
            counts["updated"] += 1
        else:
            counts["skipped"] += 1
    return counts
