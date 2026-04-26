"""Tests for Discord reader module."""

from alt_discord.reader import timestamp_to_snowflake, format_messages


def test_timestamp_to_snowflake_jst():
    """JST timestamp converts to valid snowflake."""
    result = timestamp_to_snowflake("2026-01-01T00:00:00+09:00")
    assert isinstance(result, str)
    assert int(result) > 0


def test_timestamp_to_snowflake_utc_matches_jst():
    """Same instant in UTC and JST produces the same snowflake."""
    result_jst = timestamp_to_snowflake("2026-01-01T09:00:00+09:00")
    result_utc = timestamp_to_snowflake("2026-01-01T00:00:00+00:00")
    assert result_jst == result_utc


def test_format_messages_empty():
    assert format_messages([]) == ""


def test_format_messages_filters_empty_content():
    messages = [
        {"timestamp": "2026-01-01T12:00:00+00:00", "author": {"username": "user1"}, "content": ""},
        {"timestamp": "2026-01-01T12:01:00+00:00", "author": {"username": "user1"}, "content": "test memo"},
    ]
    result = format_messages(messages)
    assert "test memo" in result
    assert result.count("\n") == 0  # only one non-empty message


def test_format_messages_sorted_by_time():
    messages = [
        {"timestamp": "2026-01-01T12:05:00+00:00", "author": {"username": "a"}, "content": "second"},
        {"timestamp": "2026-01-01T12:00:00+00:00", "author": {"username": "b"}, "content": "first"},
    ]
    result = format_messages(messages)
    lines = result.strip().split("\n")
    assert "first" in lines[0]
    assert "second" in lines[1]
