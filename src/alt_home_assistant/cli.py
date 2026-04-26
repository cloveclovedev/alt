"""CLI entry point for alt-home-assistant."""

import argparse
import json
import sys

from dotenv import load_dotenv

from alt_db.connection import NeonHTTP
from alt_db import config

from .client import HomeAssistantClient


def build_parser() -> argparse.ArgumentParser:
    """Build the argument parser."""
    parser = argparse.ArgumentParser(
        prog="alt-home-assistant",
        description="Home Assistant REST API client",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    # tts
    tts_parser = subparsers.add_parser("tts", help="Send TTS message to a media player")
    tts_parser.add_argument("message", help="Text to speak")
    tts_parser.add_argument("--entity", help="media_player entity ID (default: from config)")

    # state
    state_parser = subparsers.add_parser("state", help="Get entity state")
    state_parser.add_argument("entity_id", help="Entity ID to query")

    # call
    call_parser = subparsers.add_parser("call", help="Call a HA service")
    call_parser.add_argument("domain", help="Service domain (e.g., light)")
    call_parser.add_argument("service", help="Service name (e.g., turn_on)")
    call_parser.add_argument("--data", help="JSON data for the service call")

    return parser


def main():
    load_dotenv()
    parser = build_parser()
    args = parser.parse_args()

    client = HomeAssistantClient.from_env()

    try:
        if args.command == "tts":
            entity = args.entity
            if not entity:
                db = NeonHTTP.from_env()
                entity = config.get(db, "home_assistant.tts_entity")
            if not entity:
                print("Error: --entity required (or set home_assistant.tts_entity in config)", file=sys.stderr)
                sys.exit(1)
            client.tts(args.message, entity_id=entity)
            print(json.dumps({"status": "sent", "entity": entity, "message": args.message}))

        elif args.command == "state":
            result = client.get_state(args.entity_id)
            print(json.dumps(result, default=str))

        elif args.command == "call":
            data = json.loads(args.data) if args.data else {}
            result = client.call_service(args.domain, args.service, data)
            print(json.dumps(result, default=str))

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
