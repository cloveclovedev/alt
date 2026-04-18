"""Entry CRUD operations."""

import json

from .connection import NeonHTTP


def _next_param(params: list, value) -> str:
    """Append value to params list and return $N placeholder."""
    params.append(value)
    return f"${len(params)}"


def add_entry(
    db: NeonHTTP,
    type: str,
    title: str,
    content: str | None = None,
    status: str | None = None,
    tags: list[str] | None = None,
    metadata: dict | None = None,
) -> str:
    """Create a new entry. Returns the entry ID."""
    result = db.execute(
        """
        INSERT INTO entries (type, title, content, status, tags, metadata)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
        """,
        [type, title, content, status, json.dumps(tags or []), json.dumps(metadata or {})],
    )
    return str(result.rows[0][0])


def get_entry(db: NeonHTTP, entry_id: str) -> dict | None:
    """Get a single entry by ID."""
    result = db.execute(
        "SELECT id, type, title, content, status, tags, metadata, created_at, updated_at FROM entries WHERE id = $1",
        [entry_id],
    )
    if not result.rows:
        return None
    return _row_to_dict(result.rows[0])


def list_entries(
    db: NeonHTTP,
    type: str | None = None,
    status: str | None = None,
    since_days: int | None = None,
    due_within_days: int | None = None,
    tag: str | None = None,
) -> list[dict]:
    """List entries with optional filters."""
    conditions = []
    params: list = []

    if type is not None:
        conditions.append(f"type = {_next_param(params, type)}")
    if status is not None:
        conditions.append(f"status = {_next_param(params, status)}")
    if since_days is not None:
        conditions.append(f"created_at >= now() - make_interval(days => {_next_param(params, since_days)})")
    if due_within_days is not None:
        conditions.append(f"(metadata->>'target_date')::date <= (current_date + make_interval(days => {_next_param(params, due_within_days)}))")
        conditions.append("(metadata->>'target_date') IS NOT NULL")
    if tag is not None:
        conditions.append(f"tags @> {_next_param(params, json.dumps([tag]))}")

    where = " AND ".join(conditions) if conditions else "TRUE"
    result = db.execute(
        f"SELECT id, type, title, content, status, tags, metadata, created_at, updated_at FROM entries WHERE {where} ORDER BY created_at DESC",
        params,
    )
    return [_row_to_dict(row) for row in result.rows]


def search_entries(db: NeonHTTP, query: str) -> list[dict]:
    """Search entries by title and content (case-insensitive)."""
    escaped = query.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_")
    pattern = f"%{escaped}%"
    result = db.execute(
        "SELECT id, type, title, content, status, tags, metadata, created_at, updated_at FROM entries WHERE title ILIKE $1 OR content ILIKE $2 ORDER BY created_at DESC",
        [pattern, pattern],
    )
    return [_row_to_dict(row) for row in result.rows]


def update_entry(db: NeonHTTP, entry_id: str, **updates) -> bool:
    """Update an entry. Supported fields: title, content, status, tags, metadata. Returns True if updated."""
    allowed = {"title", "content", "status", "tags", "metadata"}
    fields = {k: v for k, v in updates.items() if k in allowed}
    if not fields:
        return False

    set_clauses = []
    params: list = []
    for key, value in fields.items():
        if key in ("tags", "metadata"):
            set_clauses.append(f"{key} = {_next_param(params, json.dumps(value))}")
        else:
            set_clauses.append(f"{key} = {_next_param(params, value)}")

    set_clauses.append("updated_at = now()")

    result = db.execute(
        f"UPDATE entries SET {', '.join(set_clauses)} WHERE id = {_next_param(params, entry_id)}",
        params,
    )
    return result.row_count > 0


def delete_entry(db: NeonHTTP, entry_id: str) -> bool:
    """Delete an entry. Returns True if deleted."""
    result = db.execute("DELETE FROM entries WHERE id = $1", [entry_id])
    return result.row_count > 0


def _row_to_dict(row) -> dict:
    """Convert a database row to a dict."""
    tags = row[5]
    metadata = row[6]
    if isinstance(tags, str):
        tags = json.loads(tags)
    if isinstance(metadata, str):
        metadata = json.loads(metadata)
    return {
        "id": str(row[0]),
        "type": row[1],
        "title": row[2],
        "content": row[3],
        "status": row[4],
        "tags": tags,
        "metadata": metadata,
        "created_at": str(row[7]),
        "updated_at": str(row[8]),
    }
