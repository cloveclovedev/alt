"""Tests for Home Assistant REST API client."""

import json
from unittest.mock import patch, MagicMock

import pytest

from alt_home_assistant.client import HomeAssistantClient


def _mock_response(data: dict, status: int = 200) -> MagicMock:
    """Create a mock urllib response."""
    mock = MagicMock()
    mock.read.return_value = json.dumps(data).encode()
    mock.status = status
    mock.__enter__ = lambda s: s
    mock.__exit__ = MagicMock(return_value=False)
    return mock


class TestHomeAssistantClient:
    def test_from_env(self):
        with patch.dict("os.environ", {"HA_URL": "http://ha.local:8123", "HA_TOKEN": "test-token"}):
            client = HomeAssistantClient.from_env()
            assert client.url == "http://ha.local:8123"
            assert client.token == "test-token"

    def test_from_env_missing_vars(self):
        with patch.dict("os.environ", {}, clear=True):
            with pytest.raises(RuntimeError, match="HA_URL"):
                HomeAssistantClient.from_env()

    def test_get_state(self):
        client = HomeAssistantClient("http://ha.local:8123", "test-token")
        response = _mock_response({"entity_id": "sensor.test", "state": "on"})
        with patch("urllib.request.urlopen", return_value=response) as mock_urlopen:
            result = client.get_state("sensor.test")
            assert result["state"] == "on"
            req = mock_urlopen.call_args[0][0]
            assert req.full_url == "http://ha.local:8123/api/states/sensor.test"
            assert req.get_header("Authorization") == "Bearer test-token"

    def test_call_service(self):
        client = HomeAssistantClient("http://ha.local:8123", "test-token")
        response = _mock_response([])
        with patch("urllib.request.urlopen", return_value=response) as mock_urlopen:
            client.call_service("tts", "speak", {"entity_id": "media_player.room", "message": "hello"})
            req = mock_urlopen.call_args[0][0]
            assert req.full_url == "http://ha.local:8123/api/services/tts/speak"
            assert req.method == "POST"
            body = json.loads(req.data)
            assert body["entity_id"] == "media_player.room"
            assert body["message"] == "hello"

    def test_tts(self):
        client = HomeAssistantClient("http://ha.local:8123", "test-token")
        response = _mock_response([])
        with patch("urllib.request.urlopen", return_value=response) as mock_urlopen:
            client.tts("おはようございます", entity_id="media_player.room")
            req = mock_urlopen.call_args[0][0]
            assert req.full_url == "http://ha.local:8123/api/services/tts/speak"
            body = json.loads(req.data)
            assert body["entity_id"] == "media_player.room"
            assert body["message"] == "おはようございます"

    def test_url_trailing_slash_stripped(self):
        client = HomeAssistantClient("http://ha.local:8123/", "test-token")
        assert client.url == "http://ha.local:8123"
