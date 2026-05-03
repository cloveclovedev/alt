"""Tests for alt-db config CLI argument parsing."""

from alt_db.cli import build_parser


def test_config_get_args():
    parser = build_parser()
    args = parser.parse_args(["config", "get", "discord.daily_channel_id"])
    assert args.command == "config"
    assert args.action == "get"
    assert args.key == "discord.daily_channel_id"


def test_config_set_args_inline_value():
    parser = build_parser()
    args = parser.parse_args(["config", "set", "wake.prep_minutes", "60"])
    assert args.action == "set"
    assert args.key == "wake.prep_minutes"
    assert args.value == "60"
    assert args.from_file is None


def test_config_set_args_from_file():
    parser = build_parser()
    args = parser.parse_args(["config", "set", "routines", "--from-file", "routines.json"])
    assert args.action == "set"
    assert args.key == "routines"
    assert args.from_file == "routines.json"


def test_config_list_args_no_prefix():
    parser = build_parser()
    args = parser.parse_args(["config", "list"])
    assert args.action == "list"
    assert args.prefix is None


def test_config_list_args_with_prefix():
    parser = build_parser()
    args = parser.parse_args(["config", "list", "--prefix", "discord."])
    assert args.action == "list"
    assert args.prefix == "discord."


def test_config_delete_args():
    parser = build_parser()
    args = parser.parse_args(["config", "delete", "x.product_links.old"])
    assert args.action == "delete"
    assert args.key == "x.product_links.old"


def test_config_set_meta_args():
    parser = build_parser()
    args = parser.parse_args(["config", "set-meta", "plan.discord.channel_id", '{"type":"string"}'])
    assert args.action == "set-meta"
    assert args.key == "plan.discord.channel_id"
    assert args.metadata == '{"type":"string"}'


def test_config_list_with_meta_flag():
    parser = build_parser()
    args = parser.parse_args(["config", "list", "--with-meta"])
    assert args.action == "list"
    assert args.with_meta is True


def test_config_list_with_meta_default_false():
    parser = build_parser()
    args = parser.parse_args(["config", "list"])
    assert args.with_meta is False


def test_config_seed_default_path():
    parser = build_parser()
    args = parser.parse_args(["config", "seed"])
    assert args.action == "seed"
    assert args.force is False
    assert args.file is None  # uses default path resolution


def test_config_seed_with_force_and_file():
    parser = build_parser()
    args = parser.parse_args(["config", "seed", "--force", "--file", "custom.yaml"])
    assert args.force is True
    assert args.file == "custom.yaml"
