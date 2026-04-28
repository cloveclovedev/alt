"""Tests for Discord CLI argument parsing."""

import json
from unittest.mock import patch

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


def test_post_thread_command_with_message_id():
    parser = build_parser()
    args = parser.parse_args([
        "post-thread", "123456789",
        "📋 2026-04-15 (Tue) Daily Plan",
        "Full plan content here",
        "--message-id", "msg-999",
    ])
    assert args.command == "post-thread"
    assert args.channel_id == "123456789"
    assert args.thread_name == "📋 2026-04-15 (Tue) Daily Plan"
    assert args.message == "Full plan content here"
    assert args.message_id == "msg-999"


def test_post_thread_command_without_message_id():
    parser = build_parser()
    args = parser.parse_args([
        "post-thread", "123456789",
        "📋 2026-04-15 (Tue) Daily Plan",
        "Hello world",
    ])
    assert args.message_id is None


@patch("alt_discord.cli.post_message")
@patch("alt_discord.cli.create_thread_from_message")
@patch.dict("os.environ", {"DISCORD_BOT_TOKEN": "test-token"})
def test_post_thread_with_message_id_skips_channel_post(mock_create_thread, mock_post_message):
    """When --message-id is given, skip channel post and create thread on that message."""
    mock_create_thread.return_value = {"id": "thread-100", "name": "Test Thread"}
    mock_post_message.return_value = ["msg-body-1"]

    from alt_discord.cli import main
    with patch("sys.argv", [
        "alt-discord", "post-thread", "ch-123",
        "📋 2026-04-15 (Tue) Daily Plan",
        "Full plan body",
        "--message-id", "msg-999",
    ]):
        with patch("builtins.print") as mock_print:
            main()

    # Thread created on the provided message, NOT a new channel post
    mock_create_thread.assert_called_once_with("ch-123", "msg-999", "📋 2026-04-15 (Tue) Daily Plan")
    # Body posted into the thread
    mock_post_message.assert_called_once_with("thread-100", "Full plan body")
    # Output includes the provided message_id
    output = json.loads(mock_print.call_args[0][0])
    assert output["message_id"] == "msg-999"
    assert output["thread_id"] == "thread-100"
