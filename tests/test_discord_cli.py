"""Tests for Discord CLI argument parsing."""

from alt_discord.cli import build_parser


def test_read_command():
    parser = build_parser()
    args = parser.parse_args(["read", "123456789"])
    assert args.command == "read"
    assert args.channel_id == "123456789"
    assert args.after is None


def test_read_command_with_after():
    parser = build_parser()
    args = parser.parse_args(["read", "123456789", "--after", "2026-04-01T00:00:00+09:00"])
    assert args.command == "read"
    assert args.after == "2026-04-01T00:00:00+09:00"


def test_post_command():
    parser = build_parser()
    args = parser.parse_args(["post", "123456789", "Hello world"])
    assert args.command == "post"
    assert args.channel_id == "123456789"
    assert args.message == "Hello world"


def test_post_thread_command():
    parser = build_parser()
    args = parser.parse_args([
        "post-thread", "123456789",
        "📋 2026-04-12 (Sat) Daily Plan",
        "Hello world",
    ])
    assert args.command == "post-thread"
    assert args.channel_id == "123456789"
    assert args.thread_name == "📋 2026-04-12 (Sat) Daily Plan"
    assert args.message == "Hello world"
