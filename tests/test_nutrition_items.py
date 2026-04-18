"""Tests for nutrition_items DB operations."""

from unittest.mock import MagicMock
from alt_db.nutrition_items import (
    add_item,
    get_item_by_name,
    list_items,
    update_item,
    delete_item,
    _row_to_dict,
)


def _make_db(rows=None, row_count=1):
    db = MagicMock()
    result = MagicMock()
    result.rows = rows or []
    result.row_count = row_count
    db.execute.return_value = result
    return db


class TestAddItem:
    def test_inserts_with_correct_sql(self):
        db = _make_db(
            rows=[("id-1", "プロテイン", 249.0, 23.0, "user_registered", "2026-04-10", "2026-04-10")],
        )
        result = add_item(db, name="プロテイン", calories=249.0, protein=23.0, source="user_registered")
        assert result["name"] == "プロテイン"
        assert result["calories_kcal"] == 249.0
        assert result["protein_g"] == 23.0
        sql = db.execute.call_args[0][0]
        assert "INSERT INTO nutrition_items" in sql


class TestGetItemByName:
    def test_returns_item_when_found(self):
        db = _make_db(
            rows=[("id-1", "プロテイン", 249.0, 23.0, "user_registered", "2026-04-10", "2026-04-10")],
        )
        result = get_item_by_name(db, "プロテイン")
        assert result is not None
        assert result["name"] == "プロテイン"

    def test_returns_none_when_not_found(self):
        db = _make_db(rows=[])
        result = get_item_by_name(db, "unknown")
        assert result is None


class TestListItems:
    def test_returns_all_items(self):
        db = _make_db(
            rows=[
                ("id-1", "プロテイン", 249.0, 23.0, "user_registered", "2026-04-10", "2026-04-10"),
                ("id-2", "EAA", 50.0, 6.2, "user_registered", "2026-04-10", "2026-04-10"),
            ],
        )
        items = list_items(db)
        assert len(items) == 2


class TestUpdateItem:
    def test_returns_true_on_success(self):
        db = _make_db(row_count=1)
        assert update_item(db, "プロテイン", calories=130.0) is True

    def test_returns_false_when_not_found(self):
        db = _make_db(row_count=0)
        assert update_item(db, "unknown", calories=100.0) is False


class TestDeleteItem:
    def test_returns_true_on_success(self):
        db = _make_db(row_count=1)
        assert delete_item(db, "プロテイン") is True

    def test_returns_false_when_not_found(self):
        db = _make_db(row_count=0)
        assert delete_item(db, "unknown") is False
