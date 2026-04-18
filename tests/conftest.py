"""Shared test fixtures."""

import pytest

from alt_db.connection import NeonHTTP


@pytest.fixture
def db():
    """Provide a NeonHTTP client with DELETE-based cleanup after each test."""
    client = NeonHTTP.from_env()
    created_entry_ids: list[str] = []
    created_routine_names: set[str] = set()
    created_body_timestamps: list[str] = []
    yield client, created_entry_ids, created_routine_names, created_body_timestamps
    # Teardown: delete test data
    for eid in created_entry_ids:
        client.execute("DELETE FROM entries WHERE id = $1", [eid])
    for name in created_routine_names:
        client.execute(
            "DELETE FROM routine_events WHERE routine_name = $1", [name]
        )
    if created_body_timestamps:
        placeholders = ", ".join(
            f"${i+1}" for i in range(len(created_body_timestamps))
        )
        client.execute(
            f"DELETE FROM body_measurements WHERE measured_at IN ({placeholders})",
            created_body_timestamps,
        )
