"""Tests for config operations."""

import json

from alt_db import config


def test_get_missing_key_returns_none(config_db):
    client, _ = config_db
    assert config.get(client, "test.config.does_not_exist") is None


def test_get_missing_key_returns_default(config_db):
    client, _ = config_db
    assert config.get(client, "test.config.does_not_exist", default="fallback") == "fallback"


def test_set_and_get_string(config_db):
    client, keys = config_db
    keys.append("test.config.string")
    config.set(client, "test.config.string", "hello")
    assert config.get(client, "test.config.string") == "hello"


def test_set_and_get_number(config_db):
    client, keys = config_db
    keys.append("test.config.number")
    config.set(client, "test.config.number", 42)
    assert config.get(client, "test.config.number") == 42


def test_set_and_get_bool(config_db):
    client, keys = config_db
    keys.append("test.config.bool")
    config.set(client, "test.config.bool", True)
    assert config.get(client, "test.config.bool") is True


def test_set_and_get_array(config_db):
    client, keys = config_db
    keys.append("test.config.array")
    config.set(client, "test.config.array", ["a", "b", "c"])
    assert config.get(client, "test.config.array") == ["a", "b", "c"]


def test_set_and_get_object(config_db):
    client, keys = config_db
    keys.append("test.config.object")
    config.set(client, "test.config.object", {"a": 1, "nested": {"b": 2}})
    assert config.get(client, "test.config.object") == {"a": 1, "nested": {"b": 2}}


def test_set_overwrites_existing(config_db):
    client, keys = config_db
    keys.append("test.config.overwrite")
    config.set(client, "test.config.overwrite", "first")
    config.set(client, "test.config.overwrite", "second")
    assert config.get(client, "test.config.overwrite") == "second"


def test_set_updates_updated_at(config_db):
    client, keys = config_db
    keys.append("test.config.timestamps")
    config.set(client, "test.config.timestamps", "v1")
    first = client.execute(
        "SELECT created_at, updated_at FROM config WHERE key = $1",
        ["test.config.timestamps"],
    ).rows[0]
    config.set(client, "test.config.timestamps", "v2")
    second = client.execute(
        "SELECT created_at, updated_at FROM config WHERE key = $1",
        ["test.config.timestamps"],
    ).rows[0]
    assert second[0] == first[0]            # created_at unchanged
    assert second[1] >= first[1]            # updated_at advanced (or equal under fast clock)


def test_set_and_get_string_that_looks_like_json(config_db):
    """A user-stored string that happens to be valid JSON should round-trip as a string."""
    client, keys = config_db
    keys.append("test.config.json_like_string")
    config.set(client, "test.config.json_like_string", '{"not": "a dict"}')
    result = config.get(client, "test.config.json_like_string")
    assert result == '{"not": "a dict"}'
    assert isinstance(result, str)


def test_list_configs_returns_all(config_db):
    client, keys = config_db
    keys.extend(["test.list.a", "test.list.b"])
    config.set(client, "test.list.a", 1)
    config.set(client, "test.list.b", 2)
    results = config.list_configs(client)
    found = {r["key"]: r["value"] for r in results if r["key"].startswith("test.list.")}
    assert found == {"test.list.a": 1, "test.list.b": 2}


def test_list_configs_with_prefix(config_db):
    client, keys = config_db
    keys.extend(["test.prefix.x.one", "test.prefix.x.two", "test.prefix.y.one"])
    config.set(client, "test.prefix.x.one", "x1")
    config.set(client, "test.prefix.x.two", "x2")
    config.set(client, "test.prefix.y.one", "y1")
    results = config.list_configs(client, prefix="test.prefix.x.")
    found_keys = {r["key"] for r in results}
    assert "test.prefix.x.one" in found_keys
    assert "test.prefix.x.two" in found_keys
    assert "test.prefix.y.one" not in found_keys


def test_list_configs_includes_timestamps(config_db):
    client, keys = config_db
    keys.append("test.list.with_ts")
    config.set(client, "test.list.with_ts", "value")
    results = config.list_configs(client, prefix="test.list.with_ts")
    assert len(results) == 1
    row = results[0]
    assert "created_at" in row
    assert "updated_at" in row


def test_delete_existing_returns_true(config_db):
    client, _ = config_db
    config.set(client, "test.delete.target", "value")
    # No need to track in keys — delete handles it
    assert config.delete(client, "test.delete.target") is True
    assert config.get(client, "test.delete.target") is None


def test_delete_missing_returns_false(config_db):
    client, _ = config_db
    assert config.delete(client, "test.delete.never_existed") is False


def test_set_meta_creates_row_when_key_absent(config_db):
    client, keys = config_db
    keys.append("test.config.meta_only")
    config.set_meta(client, "test.config.meta_only", {"type": "string", "description": "x"})
    rows = client.execute(
        "SELECT value::text, metadata::text FROM config WHERE key = $1",
        ["test.config.meta_only"],
    ).rows
    assert len(rows) == 1
    assert rows[0][0] == "null"  # value defaults to JSON null
    assert json.loads(rows[0][1]) == {"type": "string", "description": "x"}


def test_set_meta_updates_existing_row_metadata_only(config_db):
    client, keys = config_db
    keys.append("test.config.meta_update")
    config.set(client, "test.config.meta_update", "real-value")
    config.set_meta(client, "test.config.meta_update", {"type": "string", "description": "y"})
    assert config.get(client, "test.config.meta_update") == "real-value"
    meta_row = client.execute(
        "SELECT metadata::text FROM config WHERE key = $1",
        ["test.config.meta_update"],
    ).rows[0]
    assert json.loads(meta_row[0]) == {"type": "string", "description": "y"}


def test_list_with_meta_returns_metadata(config_db):
    client, keys = config_db
    keys.extend(["test.lwm.a", "test.lwm.b"])
    config.set(client, "test.lwm.a", 1)
    config.set_meta(client, "test.lwm.a", {"type": "number"})
    config.set(client, "test.lwm.b", "v")
    results = config.list_with_meta(client, prefix="test.lwm.")
    found = {r["key"]: r for r in results}
    assert found["test.lwm.a"]["value"] == 1
    assert found["test.lwm.a"]["metadata"] == {"type": "number"}
    assert found["test.lwm.b"]["value"] == "v"
    assert found["test.lwm.b"]["metadata"] == {}
