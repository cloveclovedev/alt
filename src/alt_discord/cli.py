"""CLI entry point for alt-discord."""

import argparse
import json

from dotenv import load_dotenv

from .reader import fetch_messages, format_messages
from .poster import post_message, split_message, create_thread_from_message


def build_parser() -> argparse.ArgumentParser:
    """Build the argument parser."""
    parser = argparse.ArgumentParser(prog="alt-discord", description="Discord channel read/post utilities")
    subparsers = parser.add_subparsers(dest="command", required=True)

    # read
    read_parser = subparsers.add_parser("read")
    read_parser.add_argument("channel_id")
    read_parser.add_argument("--after", help="ISO 8601 timestamp to filter messages after")

    # post
    post_parser = subparsers.add_parser("post")
    post_parser.add_argument("channel_id")
    post_parser.add_argument("message")

    # post-thread
    pt_parser = subparsers.add_parser("post-thread")
    pt_parser.add_argument("channel_id")
    pt_parser.add_argument("thread_name")
    pt_parser.add_argument("message")
    pt_parser.add_argument("--message-id", default=None, help="Existing message ID to create thread on (skip channel post)")

    return parser


def main():
    load_dotenv()
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "read":
        messages = fetch_messages(args.channel_id, after_timestamp=args.after)
        output = format_messages(messages)
        if output:
            print(output)

    elif args.command == "post":
        message_ids = post_message(args.channel_id, args.message)
        print(json.dumps({"message_ids": message_ids, "chunks": len(message_ids)}))

    elif args.command == "post-thread":
        if args.message_id:
            # Create thread on an existing message, post body into the thread
            thread = create_thread_from_message(
                args.channel_id, args.message_id, args.thread_name
            )
            thread_id = thread["id"]
            chunks = split_message(args.message)
            for chunk in chunks:
                post_message(thread_id, chunk)
            print(json.dumps({
                "message_id": args.message_id,
                "thread_id": thread_id,
                "chunks": len(chunks),
            }))
        else:
            # Original behavior: post first chunk to channel, create thread, post rest
            chunks = split_message(args.message)
            first_ids = post_message(args.channel_id, chunks[0])
            first_message_id = first_ids[0]
            thread = create_thread_from_message(
                args.channel_id, first_message_id, args.thread_name
            )
            thread_id = thread["id"]
            for chunk in chunks[1:]:
                post_message(thread_id, chunk)
            print(json.dumps({
                "message_id": first_message_id,
                "thread_id": thread_id,
                "chunks": len(chunks),
            }))


if __name__ == "__main__":
    main()
