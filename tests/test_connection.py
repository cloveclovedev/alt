"""Tests for NeonHTTP client."""

from alt_db.connection import NeonHTTP, QueryResult


def test_from_env_creates_client():
    db = NeonHTTP.from_env()
    assert db.host
    assert db.database
    assert db.user
    assert db.password


def test_execute_simple_query():
    db = NeonHTTP.from_env()
    result = db.execute("SELECT 1 AS n")
    assert isinstance(result, QueryResult)
    assert len(result.rows) == 1
    assert result.rows[0][0] == 1
    assert result.row_count == 1


def test_execute_with_params():
    db = NeonHTTP.from_env()
    result = db.execute("SELECT $1::text AS greeting", ["hello"])
    assert result.rows[0][0] == "hello"


def test_execute_no_rows():
    db = NeonHTTP.from_env()
    result = db.execute("SELECT 1 WHERE false")
    assert result.rows == []
    assert result.row_count == 0
