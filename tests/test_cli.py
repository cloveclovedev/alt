"""Tests for CLI argument parsing and output formatting."""

from alt_db.cli import build_parser, format_entry, format_routine_event


def test_entry_add_args():
    parser = build_parser()
    args = parser.parse_args(["entry", "add", "--type", "memo", "--title", "Test memo"])
    assert args.command == "entry"
    assert args.action == "add"
    assert args.type == "memo"
    assert args.title == "Test memo"


def test_entry_add_with_tags():
    parser = build_parser()
    args = parser.parse_args(["entry", "add", "--type", "goal", "--title", "Goal", "--tags", '["peppercheck"]'])
    assert args.tags == '["peppercheck"]'


def test_entry_list_args():
    parser = build_parser()
    args = parser.parse_args(["entry", "list", "--type", "goal", "--status", "active"])
    assert args.type == "goal"
    assert args.status == "active"


def test_entry_list_since():
    parser = build_parser()
    args = parser.parse_args(["entry", "list", "--since", "7d"])
    assert args.since == "7d"


def test_entry_search_args():
    parser = build_parser()
    args = parser.parse_args(["entry", "search", "Flutter"])
    assert args.query == "Flutter"


def test_entry_update_args():
    parser = build_parser()
    args = parser.parse_args(["entry", "update", "abc-123", "--status", "achieved"])
    assert args.id == "abc-123"
    assert args.status == "achieved"


def test_entry_delete_args():
    parser = build_parser()
    args = parser.parse_args(["entry", "delete", "abc-123"])
    assert args.id == "abc-123"


def test_routine_complete_args():
    parser = build_parser()
    args = parser.parse_args(["routine", "complete", "Clean the toilet", "household"])
    assert args.command == "routine"
    assert args.action == "complete"
    assert args.name == "Clean the toilet"
    assert args.category == "household"


def test_routine_baseline_args():
    parser = build_parser()
    args = parser.parse_args(["routine", "baseline", "Take flea medicine", "dog", "--date", "2026-05-01"])
    assert args.date == "2026-05-01"


def test_routine_complete_with_note():
    parser = build_parser()
    args = parser.parse_args(["routine", "complete", "Dental checkup", "health", "--note", "Next: Oct 2026"])
    assert args.note == "Next: Oct 2026"


def test_format_entry():
    entry = {
        "id": "abc-123",
        "type": "goal",
        "title": "Launch app",
        "status": "active",
        "tags": ["peppercheck"],
        "created_at": "2026-04-08 10:00:00+09:00",
    }
    output = format_entry(entry)
    assert "goal" in output
    assert "Launch app" in output
    assert "active" in output
    assert "peppercheck" in output


def test_format_routine_event():
    event = {
        "routine_name": "Clean the toilet",
        "category": "household",
        "completed_at": "2026-04-08 10:00:00+09:00",
        "kind": "completed",
        "note": None,
    }
    output = format_routine_event(event)
    assert "Clean the toilet" in output
    assert "household" in output
    assert "completed" in output
