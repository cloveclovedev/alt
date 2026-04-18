"""Tests for entry operations."""

from alt_db.entries import (
    add_entry,
    delete_entry,
    get_entry,
    list_entries,
    search_entries,
    update_entry,
)


def test_add_and_get_entry(db):
    client, entry_ids, _, _ = db
    entry_id = add_entry(client, type="memo", title="Test: Rust is interesting")
    entry_ids.append(entry_id)
    result = get_entry(client, entry_id)
    assert result is not None
    assert result["type"] == "memo"
    assert result["title"] == "Test: Rust is interesting"


def test_add_entry_with_all_fields(db):
    client, entry_ids, _, _ = db
    entry_id = add_entry(
        client,
        type="goal",
        title="Test: peppercheck Android launch",
        content="Launch v1.0.0 on Google Play",
        status="active",
        tags=["peppercheck", "phase1"],
        metadata={"target_date": "2026-04-30", "revenue_target": 50000},
    )
    entry_ids.append(entry_id)
    result = get_entry(client, entry_id)
    assert result["status"] == "active"
    assert result["tags"] == ["peppercheck", "phase1"]
    assert result["metadata"]["target_date"] == "2026-04-30"


def test_list_entries_by_type(db):
    client, entry_ids, _, _ = db
    id1 = add_entry(client, type="memo", title="Test: Memo 1")
    id2 = add_entry(client, type="goal", title="Test: Goal 1")
    id3 = add_entry(client, type="memo", title="Test: Memo 2")
    entry_ids.extend([id1, id2, id3])
    results = list_entries(client, type="memo")
    result_ids = [r["id"] for r in results]
    assert id1 in result_ids
    assert id3 in result_ids
    assert id2 not in result_ids
    assert all(r["type"] == "memo" for r in results)


def test_list_entries_by_status(db):
    client, entry_ids, _, _ = db
    id1 = add_entry(client, type="goal", title="Test: Active goal", status="active")
    id2 = add_entry(client, type="goal", title="Test: Done goal", status="achieved")
    entry_ids.extend([id1, id2])
    results = list_entries(client, type="goal", status="active")
    result_ids = [r["id"] for r in results]
    assert id1 in result_ids
    assert id2 not in result_ids


def test_list_entries_by_tag(db):
    client, entry_ids, _, _ = db
    id1 = add_entry(client, type="memo", title="Test: Flutter memo", tags=["flutter"])
    id2 = add_entry(client, type="memo", title="Test: Rust memo", tags=["rust"])
    entry_ids.extend([id1, id2])
    results = list_entries(client, tag="flutter")
    result_ids = [r["id"] for r in results]
    assert id1 in result_ids
    assert id2 not in result_ids


def test_list_entries_since_days(db):
    client, entry_ids, _, _ = db
    entry_id = add_entry(client, type="memo", title="Test: Recent memo")
    entry_ids.append(entry_id)
    results = list_entries(client, since_days=7)
    assert any(r["id"] == entry_id for r in results)


def test_search_entries(db):
    client, entry_ids, _, _ = db
    id1 = add_entry(
        client,
        type="knowledge",
        title="Test: Flutter state management",
        content="Use Riverpod for state management in Flutter apps",
    )
    id2 = add_entry(client, type="memo", title="Test: Grocery list")
    entry_ids.extend([id1, id2])
    results = search_entries(client, "Flutter")
    result_ids = [r["id"] for r in results]
    assert id1 in result_ids


def test_update_entry(db):
    client, entry_ids, _, _ = db
    entry_id = add_entry(client, type="goal", title="Test: Launch app", status="active")
    entry_ids.append(entry_id)
    update_entry(client, entry_id, status="achieved")
    result = get_entry(client, entry_id)
    assert result["status"] == "achieved"


def test_update_entry_title_and_tags(db):
    client, entry_ids, _, _ = db
    entry_id = add_entry(client, type="memo", title="Test: Old title", tags=["old"])
    entry_ids.append(entry_id)
    update_entry(client, entry_id, title="Test: New title", tags=["new", "updated"])
    result = get_entry(client, entry_id)
    assert result["title"] == "Test: New title"
    assert result["tags"] == ["new", "updated"]


def test_delete_entry(db):
    client, entry_ids, _, _ = db
    entry_id = add_entry(client, type="memo", title="Test: To be deleted")
    # No need to track — delete_entry handles removal
    delete_entry(client, entry_id)
    result = get_entry(client, entry_id)
    assert result is None


def test_get_entry_not_found(db):
    client, _, _, _ = db
    result = get_entry(client, "00000000-0000-0000-0000-000000000000")
    assert result is None
