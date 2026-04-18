"""Tests for nutrition_targets DB operations."""

from unittest.mock import MagicMock
from alt_db.nutrition_targets import add_target, get_active_target, deactivate_target


def _make_db(rows=None, row_count=1):
    db = MagicMock()
    result = MagicMock()
    result.rows = rows or []
    result.row_count = row_count
    db.execute.return_value = result
    return db


class TestAddTarget:
    def test_deactivates_previous_and_inserts(self):
        db = _make_db(
            rows=[("id-1", 2675.0, 146.0, "2026-04-10", None, "73kg x 2.0", "ts", "ts")],
        )
        result = add_target(
            db, calories=2675.0, protein=146.0,
            effective_from="2026-04-10", rationale="73kg x 2.0 = 146g",
        )
        assert result["calories_kcal"] == 2675.0
        # Should have called execute twice: UPDATE (deactivate) + INSERT
        assert db.execute.call_count == 2


class TestGetActiveTarget:
    def test_returns_target_when_exists(self):
        db = _make_db(
            rows=[("id-1", 2675.0, 146.0, "2026-04-10", None, "73kg x 2.0", "ts", "ts")],
        )
        result = get_active_target(db, "2026-04-10")
        assert result is not None
        assert result["protein_g"] == 146.0

    def test_returns_none_when_no_target(self):
        db = _make_db(rows=[])
        result = get_active_target(db, "2026-04-10")
        assert result is None
