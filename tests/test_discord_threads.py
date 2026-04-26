"""Tests for Discord thread functions."""

import json
from unittest.mock import patch, MagicMock

from alt_discord.poster import create_thread, create_thread_from_message
from alt_discord.reader import fetch_channel_threads, get_image_urls


class TestCreateThread:
    @patch("alt_discord.poster.urllib.request.urlopen")
    @patch.dict("os.environ", {"DISCORD_BOT_TOKEN": "test-token"})
    def test_creates_thread_and_posts_initial_message(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"id": "thread-123", "name": "🍽 2026-04-10 食事記録"}
        ).encode()
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        result = create_thread(
            channel_id="ch-456",
            name="🍽 2026-04-10 食事記録",
            initial_message="今日の目標: 2,675kcal / P: 146.0g",
        )

        assert result["id"] == "thread-123"
        call_args = mock_urlopen.call_args
        req = call_args[0][0]
        assert "/channels/ch-456/threads" in req.full_url
        body = json.loads(req.data.decode())
        assert body["name"] == "🍽 2026-04-10 食事記録"
        assert "今日の目標" in body["message"]["content"]


class TestCreateThreadFromMessage:
    @patch("alt_discord.poster.urllib.request.urlopen")
    @patch.dict("os.environ", {"DISCORD_BOT_TOKEN": "test-token"})
    def test_creates_thread_on_existing_message(self, mock_urlopen):
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"id": "thread-789", "name": "📋 2026-04-12 (Sat) Daily Plan"}
        ).encode()
        mock_resp.__enter__ = lambda s: s
        mock_resp.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_resp

        result = create_thread_from_message(
            channel_id="ch-456",
            message_id="msg-123",
            name="📋 2026-04-12 (Sat) Daily Plan",
        )

        assert result["id"] == "thread-789"
        call_args = mock_urlopen.call_args
        req = call_args[0][0]
        assert "/channels/ch-456/messages/msg-123/threads" in req.full_url
        body = json.loads(req.data.decode())
        assert body["name"] == "📋 2026-04-12 (Sat) Daily Plan"
        assert body["auto_archive_duration"] == 1440


class TestFetchChannelThreads:
    @patch("alt_discord.reader._discord_get")
    def test_returns_active_threads(self, mock_get):
        mock_get.return_value = {
            "threads": [
                {"id": "t1", "name": "🍽 2026-04-10 食事記録"},
                {"id": "t2", "name": "🍽 2026-04-09 食事記録"},
            ]
        }

        threads = fetch_channel_threads("guild-789")
        assert len(threads) == 2
        assert threads[0]["name"] == "🍽 2026-04-10 食事記録"


class TestGetImageUrls:
    def test_extracts_attachment_urls(self):
        message = {
            "attachments": [
                {"url": "https://cdn.discord.com/img1.jpg", "content_type": "image/jpeg"},
                {"url": "https://cdn.discord.com/doc.pdf", "content_type": "application/pdf"},
                {"url": "https://cdn.discord.com/img2.png", "content_type": "image/png"},
            ]
        }
        urls = get_image_urls(message)
        assert urls == [
            "https://cdn.discord.com/img1.jpg",
            "https://cdn.discord.com/img2.png",
        ]

    def test_returns_empty_for_no_attachments(self):
        assert get_image_urls({"attachments": []}) == []
        assert get_image_urls({}) == []
