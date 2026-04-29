"""Tests for config seed: load_yaml_defaults + seed."""

import json
import textwrap
from pathlib import Path

import pytest

from alt_db import config


def test_load_yaml_defaults_parses_params(tmp_path: Path):
    yaml_file = tmp_path / "defaults.yaml"
    yaml_file.write_text(textwrap.dedent("""
        params:
          plan.discord.channel_id:
            type: string
            description: Discord channel
            consumed_by: [daily-plan, weekly-plan]
          daily_plan.cloud.enabled:
            type: boolean
            description: |
              Enable fallback.
            default: true
    """))
    result = config.load_yaml_defaults(str(yaml_file))
    assert "plan.discord.channel_id" in result
    assert result["plan.discord.channel_id"]["type"] == "string"
    assert result["plan.discord.channel_id"]["consumed_by"] == ["daily-plan", "weekly-plan"]
    assert result["daily_plan.cloud.enabled"]["default"] is True


def test_load_yaml_defaults_rejects_malformed(tmp_path: Path):
    yaml_file = tmp_path / "bad.yaml"
    yaml_file.write_text("params: [not, a, mapping]\n")
    with pytest.raises(ValueError, match="params must be a mapping"):
        config.load_yaml_defaults(str(yaml_file))


def test_load_yaml_defaults_requires_type(tmp_path: Path):
    yaml_file = tmp_path / "no_type.yaml"
    yaml_file.write_text(textwrap.dedent("""
        params:
          some.key:
            description: missing type
    """))
    with pytest.raises(ValueError, match="some.key.*type"):
        config.load_yaml_defaults(str(yaml_file))


def test_seed_inserts_missing_key_with_default(config_db, tmp_path: Path):
    client, keys = config_db
    keys.append("test.seed.new_key")
    yaml_file = tmp_path / "defaults.yaml"
    yaml_file.write_text(textwrap.dedent("""
        params:
          test.seed.new_key:
            type: boolean
            description: A test key
            default: true
    """))
    counts = config.seed(client, str(yaml_file))
    assert counts == {"inserted": 1, "updated": 0, "skipped": 0}
    assert config.get(client, "test.seed.new_key") is True
    rows = client.execute(
        "SELECT metadata::text FROM config WHERE key = $1",
        ["test.seed.new_key"],
    ).rows
    assert json.loads(rows[0][0])["description"] == "A test key"


def test_seed_inserts_missing_key_without_default_uses_null(config_db, tmp_path: Path):
    client, keys = config_db
    keys.append("test.seed.no_default")
    yaml_file = tmp_path / "defaults.yaml"
    yaml_file.write_text(textwrap.dedent("""
        params:
          test.seed.no_default:
            type: string
            description: No default
    """))
    counts = config.seed(client, str(yaml_file))
    assert counts == {"inserted": 1, "updated": 0, "skipped": 0}
    assert config.get(client, "test.seed.no_default") is None


def test_seed_skips_existing_row_without_force(config_db, tmp_path: Path):
    client, keys = config_db
    keys.append("test.seed.exists")
    config.set(client, "test.seed.exists", "user-value")
    config.set_meta(client, "test.seed.exists", {"type": "string", "description": "old"})
    yaml_file = tmp_path / "defaults.yaml"
    yaml_file.write_text(textwrap.dedent("""
        params:
          test.seed.exists:
            type: string
            description: new description
    """))
    counts = config.seed(client, str(yaml_file), force=False)
    assert counts == {"inserted": 0, "updated": 0, "skipped": 1}
    # value untouched, metadata untouched
    assert config.get(client, "test.seed.exists") == "user-value"
    rows = client.execute(
        "SELECT metadata::text FROM config WHERE key = $1",
        ["test.seed.exists"],
    ).rows
    assert json.loads(rows[0][0])["description"] == "old"


def test_seed_force_updates_metadata_only_preserves_value(config_db, tmp_path: Path):
    client, keys = config_db
    keys.append("test.seed.force")
    config.set(client, "test.seed.force", "user-value")
    config.set_meta(client, "test.seed.force", {"type": "string", "description": "old"})
    yaml_file = tmp_path / "defaults.yaml"
    yaml_file.write_text(textwrap.dedent("""
        params:
          test.seed.force:
            type: string
            description: new description
            default: "ignored-on-existing-row"
    """))
    counts = config.seed(client, str(yaml_file), force=True)
    assert counts == {"inserted": 0, "updated": 1, "skipped": 0}
    assert config.get(client, "test.seed.force") == "user-value"  # preserved!
    rows = client.execute(
        "SELECT metadata::text FROM config WHERE key = $1",
        ["test.seed.force"],
    ).rows
    assert json.loads(rows[0][0])["description"] == "new description"


def test_seed_does_not_touch_keys_absent_from_yaml(config_db, tmp_path: Path):
    client, keys = config_db
    keys.append("test.seed.personal")
    config.set(client, "test.seed.personal", "private")
    yaml_file = tmp_path / "defaults.yaml"
    yaml_file.write_text(textwrap.dedent("""
        params:
          test.seed.unrelated:
            type: string
            description: Unrelated
    """))
    keys.append("test.seed.unrelated")
    counts = config.seed(client, str(yaml_file), force=True)
    assert counts == {"inserted": 1, "updated": 0, "skipped": 0}
    assert config.get(client, "test.seed.personal") == "private"
