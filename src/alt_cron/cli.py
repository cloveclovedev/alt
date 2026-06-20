"""CLI entry point for alt-cron."""

import argparse
import json
import sys

from alt_cron.compute import compute_target


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="alt-cron",
        description="Cron computation utilities for cloud-scheduler",
    )
    sub = parser.add_subparsers(dest="command", required=True)

    compute = sub.add_parser(
        "compute",
        help="Compute target cron from config rows on stdin",
    )
    compute.add_argument(
        "--cron-minute",
        type=int,
        required=True,
        help="Minute slot for the unified trigger (0-59)",
    )
    return parser


def main() -> None:
    args = build_parser().parse_args()
    try:
        if args.command == "compute":
            rows = json.loads(sys.stdin.read())
            if not isinstance(rows, list):
                raise ValueError("stdin JSON must be a list of config rows")
            result = compute_target(rows, cron_minute=args.cron_minute)
            print(json.dumps(result))
    except (ValueError, json.JSONDecodeError) as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
