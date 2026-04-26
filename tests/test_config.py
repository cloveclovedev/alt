"""Tests for config operations."""

from alt_db import config


def test_get_missing_key_returns_none(config_db):
    client, _ = config_db
    assert config.get(client, "test.config.does_not_exist") is None


def test_get_missing_key_returns_default(config_db):
    client, _ = config_db
    assert config.get(client, "test.config.does_not_exist", default="fallback") == "fallback"
