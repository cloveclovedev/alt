"""Shared test fixtures."""

import pytest

from alt_db.connection import NeonHTTP


@pytest.fixture
def db():
    """Provide a NeonHTTP client with DELETE-based cleanup after each test."""
    client = NeonHTTP.from_env()
    created_entry_ids: list[str] = []
    yield client, created_entry_ids
    # Teardown: delete test entries
    for eid in created_entry_ids:
        client.execute("DELETE FROM entries WHERE id = $1", [eid])


@pytest.fixture
def config_db():
    """Provide a NeonHTTP client with config-key cleanup after each test."""
    client = NeonHTTP.from_env()
    created_keys: list[str] = []
    yield client, created_keys
    for key in created_keys:
        client.execute("DELETE FROM config WHERE key = $1", [key])
