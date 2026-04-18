"""Tests for nutrition_logs DB operations."""

from unittest.mock import MagicMock
from alt_db.nutrition_logs import add_log, list_logs_by_date, is_message_processed, daily_summary


def _make_db(rows=None, row_count=1):
    db = MagicMock()
    result = MagicMock()
    result.rows = rows or []
    result.row_count = row_count
    db.execute.return_value = result
    return db


class TestAddLog:
    def test_inserts_log(self):
        db = _make_db(
            rows=[("id-1", "2026-04-10", "breakfast", "プロテイン", 249.0, 23.0, False, "msg-1", "item_lookup", "ts", "ts")],
        )
        result = add_log(
            db, logged_date="2026-04-10", meal_type="breakfast",
            description="プロテイン", calories=249.0, protein=23.0,
            source_message_id="msg-1", estimated_by="item_lookup",
        )
        assert result["meal_type"] == "breakfast"


class TestIsMessageProcessed:
    def test_returns_true_when_exists(self):
        db = _make_db(rows=[(1,)])
        assert is_message_processed(db, "msg-1") is True

    def test_returns_false_when_not_exists(self):
        db = _make_db(rows=[(0,)])
        assert is_message_processed(db, "msg-999") is False


class TestDailySummary:
    def test_aggregates_daily_totals(self):
        db = _make_db(rows=[(2100.0, 120.5, 1)])
        result = daily_summary(db, "2026-04-10")
        assert result["total_calories"] == 2100.0
        assert result["total_protein"] == 120.5
        assert result["supplement_taken"] is True
