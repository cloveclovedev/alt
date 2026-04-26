"""Tests for CLI argument parsing and output formatting."""

from alt_db.cli import build_parser, format_entry


def test_entry_add_args():
    parser = build_parser()
    args = parser.parse_args(["entry", "add", "--type", "memo", "--title", "Test memo"])
    assert args.command == "entry"
    assert args.action == "add"
    assert args.type == "memo"
    assert args.title == "Test memo"


def test_entry_add_with_parent_id():
    parser = build_parser()
    args = parser.parse_args(["entry", "add", "--type", "memo", "--title", "Child", "--parent-id", "abc-123"])
    assert args.parent_id == "abc-123"


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


def test_entry_update_with_parent_id():
    parser = build_parser()
    args = parser.parse_args(["entry", "update", "abc-123", "--parent-id", "parent-456"])
    assert args.parent_id == "parent-456"


def test_entry_delete_args():
    parser = build_parser()
    args = parser.parse_args(["entry", "delete", "abc-123"])
    assert args.id == "abc-123"


def test_format_entry():
    entry = {
        "id": "abc-123",
        "type": "goal",
        "title": "Launch app",
        "status": "active",
        "created_at": "2026-04-08 10:00:00+09:00",
    }
    output = format_entry(entry)
    assert "goal" in output
    assert "Launch app" in output
    assert "active" in output
