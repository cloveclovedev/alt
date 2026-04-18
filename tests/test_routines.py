"""Tests for routine event operations."""

from alt_db.routines import (
    add_baseline,
    complete_routine,
    delete_event,
    get_all_last_events,
    get_history,
    get_last_event,
    update_note,
)


def test_complete_routine_and_retrieve(db):
    client, _, routine_names, _ = db
    name = "test_Clean the toilet"
    routine_names.add(name)
    complete_routine(client, name, "household")
    result = get_last_event(client, name)
    assert result is not None
    assert result["routine_name"] == name
    assert result["category"] == "household"
    assert result["kind"] == "completed"


def test_complete_routine_with_note(db):
    client, _, routine_names, _ = db
    name = "test_Dental checkup"
    routine_names.add(name)
    complete_routine(client, name, "health", note="Next: 2026-10-01")
    result = get_last_event(client, name)
    assert result["note"] == "Next: 2026-10-01"


def test_add_baseline(db):
    client, _, routine_names, _ = db
    name = "test_Take flea medicine"
    routine_names.add(name)
    add_baseline(client, name, "dog", date="2026-05-01")
    result = get_last_event(client, name)
    assert result["kind"] == "baseline"
    assert "2026-05-01" in result["completed_at"]


def test_add_baseline_with_note(db):
    client, _, routine_names, _ = db
    name = "test_Take flea medicine note"
    routine_names.add(name)
    add_baseline(client, name, "dog", date="2026-05-01", note="Start of season")
    result = get_last_event(client, name)
    assert result["note"] == "Start of season"


def test_get_last_event_returns_most_recent(db):
    client, _, routine_names, _ = db
    name = "test_Clean toilet recent"
    routine_names.add(name)
    complete_routine(client, name, "household", note="first")
    complete_routine(client, name, "household", note="second")
    result = get_last_event(client, name)
    assert result["note"] == "second"


def test_get_last_event_not_found(db):
    client, _, _, _ = db
    result = get_last_event(client, "test_Nonexistent routine")
    assert result is None


def test_get_all_last_events(db):
    client, _, routine_names, _ = db
    name1 = "test_All clean toilet"
    name2 = "test_All dental checkup"
    routine_names.update([name1, name2])
    complete_routine(client, name1, "household")
    complete_routine(client, name2, "health")
    results = get_all_last_events(client)
    names = [r["routine_name"] for r in results]
    assert name1 in names
    assert name2 in names


def test_get_all_last_events_returns_only_latest(db):
    client, _, routine_names, _ = db
    name = "test_Latest clean toilet"
    routine_names.add(name)
    complete_routine(client, name, "household", note="old")
    complete_routine(client, name, "household", note="new")
    results = get_all_last_events(client)
    matched = [r for r in results if r["routine_name"] == name]
    assert len(matched) == 1
    assert matched[0]["note"] == "new"


def test_get_history(db):
    client, _, routine_names, _ = db
    name = "test_History routine"
    routine_names.add(name)
    complete_routine(client, name, "household", note="first")
    complete_routine(client, name, "household", note="second")
    results = get_history(client, name)
    assert len(results) == 2
    assert results[0]["note"] == "second"  # newest first
    assert results[1]["note"] == "first"
    assert "id" in results[0]


def test_get_history_empty(db):
    client, _, _, _ = db
    results = get_history(client, "test_Nonexistent history")
    assert results == []


def test_delete_event(db):
    client, _, routine_names, _ = db
    name = "test_Delete routine"
    routine_names.add(name)
    complete_routine(client, name, "household", note="to delete")
    history = get_history(client, name)
    event_id = history[0]["id"]
    assert delete_event(client, event_id) is True
    assert get_history(client, name) == []


def test_delete_event_not_found(db):
    client, _, _, _ = db
    assert delete_event(client, "00000000-0000-0000-0000-000000000000") is False


def test_update_note(db):
    client, _, routine_names, _ = db
    name = "test_Update note routine"
    routine_names.add(name)
    complete_routine(client, name, "household", note="old note")
    history = get_history(client, name)
    event_id = history[0]["id"]
    assert update_note(client, event_id, "new note") is True
    result = get_last_event(client, name)
    assert result["note"] == "new note"


def test_update_note_to_null(db):
    client, _, routine_names, _ = db
    name = "test_Update note null"
    routine_names.add(name)
    complete_routine(client, name, "household", note="has note")
    history = get_history(client, name)
    event_id = history[0]["id"]
    assert update_note(client, event_id, None) is True
    result = get_last_event(client, name)
    assert result["note"] is None


def test_update_note_not_found(db):
    client, _, _, _ = db
    assert update_note(client, "00000000-0000-0000-0000-000000000000", "x") is False
