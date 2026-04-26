"""Tests for alt-home-assistant CLI argument parsing."""

from alt_home_assistant.cli import build_parser


class TestTtsCommand:
    def test_tts_basic(self):
        parser = build_parser()
        args = parser.parse_args(["tts", "おはようございます"])
        assert args.command == "tts"
        assert args.message == "おはようございます"
        assert args.entity is None

    def test_tts_with_entity(self):
        parser = build_parser()
        args = parser.parse_args(["tts", "hello", "--entity", "media_player.bedroom"])
        assert args.message == "hello"
        assert args.entity == "media_player.bedroom"


class TestStateCommand:
    def test_state_basic(self):
        parser = build_parser()
        args = parser.parse_args(["state", "sensor.temperature"])
        assert args.command == "state"
        assert args.entity_id == "sensor.temperature"


class TestCallCommand:
    def test_call_basic(self):
        parser = build_parser()
        args = parser.parse_args(["call", "light", "turn_on"])
        assert args.command == "call"
        assert args.domain == "light"
        assert args.service == "turn_on"
        assert args.data is None

    def test_call_with_data(self):
        parser = build_parser()
        args = parser.parse_args(["call", "light", "turn_on", "--data", '{"entity_id": "light.bedroom"}'])
        assert args.data == '{"entity_id": "light.bedroom"}'
