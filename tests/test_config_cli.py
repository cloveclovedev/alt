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
