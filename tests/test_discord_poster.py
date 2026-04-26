"""Tests for Discord poster module."""

from alt_discord.poster import split_message


def test_split_short_message():
    result = split_message("hello")
    assert result == ["hello"]


def test_split_long_message():
    line = "a" * 100 + "\n"
    text = line * 25  # 2525 chars
    result = split_message(text)
    assert len(result) == 2
    assert all(len(chunk) <= 2000 for chunk in result)


def test_split_preserves_content():
    text = "line1\nline2\nline3"
    result = split_message(text)
    assert result == ["line1\nline2\nline3"]


def test_split_empty():
    result = split_message("")
    assert result == [""]
