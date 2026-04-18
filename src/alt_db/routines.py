"""Routine event operations."""

from .connection import NeonHTTP


def complete_routine(db: NeonHTTP, name: str, category: str, note: str | None = None):
    """Record a routine completion."""
    db.execute(
        "INSERT INTO routine_events (routine_name, category, completed_at, kind, note) VALUES ($1, $2, clock_timestamp(), 'completed', $3)",
        [name, category, note],
    )


def add_baseline(db: NeonHTTP, name: str, category: str, date: str, note: str | None = None):
    """Record a baseline event (tracking start point)."""
    db.execute(
        "INSERT INTO routine_events (routine_name, category, completed_at, kind, note) VALUES ($1, $2, $3, 'baseline', $4)",
        [name, category, f"{date}T00:00:00+00:00", note],
    )


def get_last_event(db: NeonHTTP, name: str) -> dict | None:
    """Get the most recent event for a routine."""
    result = db.execute(
        "SELECT routine_name, category, completed_at, kind, note FROM routine_events WHERE routine_name = $1 ORDER BY completed_at DESC LIMIT 1",
        [name],
    )
    if not result.rows:
        return None
    row = result.rows[0]
    return {
        "routine_name": row[0],
        "category": row[1],
        "completed_at": str(row[2]),
        "kind": row[3],
        "note": row[4],
    }


def get_all_last_events(db: NeonHTTP) -> list[dict]:
    """Get the most recent event for each routine."""
    result = db.execute(
        """
        SELECT DISTINCT ON (routine_name)
            routine_name, category, completed_at, kind, note
        FROM routine_events
        ORDER BY routine_name, completed_at DESC
        """
    )
    return [
        {
            "routine_name": row[0],
            "category": row[1],
            "completed_at": str(row[2]),
            "kind": row[3],
            "note": row[4],
        }
        for row in result.rows
    ]


def get_history(db: NeonHTTP, name: str) -> list[dict]:
    """Get all events for a routine, newest first."""
    result = db.execute(
        "SELECT id, routine_name, category, completed_at, kind, note FROM routine_events WHERE routine_name = $1 ORDER BY completed_at DESC",
        [name],
    )
    return [
        {
            "id": str(row[0]),
            "routine_name": row[1],
            "category": row[2],
            "completed_at": str(row[3]),
            "kind": row[4],
            "note": row[5],
        }
        for row in result.rows
    ]


def delete_event(db: NeonHTTP, event_id: str) -> bool:
    """Delete a routine event by ID. Returns True if deleted."""
    result = db.execute("DELETE FROM routine_events WHERE id = $1", [event_id])
    return result.row_count > 0


def update_note(db: NeonHTTP, event_id: str, note: str | None) -> bool:
    """Update the note of a routine event. Returns True if updated."""
    result = db.execute(
        "UPDATE routine_events SET note = $1 WHERE id = $2",
        [note, event_id],
    )
    return result.row_count > 0
