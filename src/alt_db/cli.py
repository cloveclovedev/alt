"""CLI entry point for alt-db."""

import argparse
import json
import os
import sys

from .connection import NeonHTTP
from . import entries


def _default_seed_path() -> str:
    """Resolve the default config-defaults.yaml path.

    Walks up from CWD looking for .claude/config-defaults.yaml, falling back
    to a path relative to CWD if not found.
    """
    cwd = os.getcwd()
    cur = cwd
    while True:
        candidate = os.path.join(cur, ".claude", "config-defaults.yaml")
        if os.path.exists(candidate):
            return candidate
        parent = os.path.dirname(cur)
        if parent == cur:
            break
        cur = parent
    return os.path.join(cwd, ".claude", "config-defaults.yaml")


def build_parser() -> argparse.ArgumentParser:
    """Build the argument parser."""
    parser = argparse.ArgumentParser(prog="alt-db", description="alt second brain knowledge store")
    parser.add_argument("--json", action="store_true", help="Output as JSON")
    subparsers = parser.add_subparsers(dest="command", required=True)

    # --- entry ---
    entry_parser = subparsers.add_parser("entry")
    entry_sub = entry_parser.add_subparsers(dest="action", required=True)

    # entry add
    entry_add = entry_sub.add_parser("add")
    entry_add.add_argument("--type", required=True)
    entry_add.add_argument("--title", required=True)
    entry_add.add_argument("--content")
    entry_add.add_argument("--status")
    entry_add.add_argument("--parent-id", dest="parent_id")
    entry_add.add_argument("--metadata")

    # entry list
    entry_list = entry_sub.add_parser("list")
    entry_list.add_argument("--type")
    entry_list.add_argument("--status")
    entry_list.add_argument("--since")
    entry_list.add_argument("--due-within", dest="due_within")

    # entry search
    entry_search = entry_sub.add_parser("search")
    entry_search.add_argument("query")

    # entry update
    entry_update = entry_sub.add_parser("update")
    entry_update.add_argument("id")
    entry_update.add_argument("--title")
    entry_update.add_argument("--content")
    entry_update.add_argument("--status")
    entry_update.add_argument("--parent-id", dest="parent_id")
    entry_update.add_argument("--metadata")

    # entry delete
    entry_delete = entry_sub.add_parser("delete")
    entry_delete.add_argument("id")

    # --- config ---
    config_parser = subparsers.add_parser("config")
    config_sub = config_parser.add_subparsers(dest="action", required=True)

    # config get
    config_get = config_sub.add_parser("get")
    config_get.add_argument("key")

    # config set
    config_set = config_sub.add_parser("set")
    config_set.add_argument("key")
    config_set.add_argument("value", nargs="?", default=None)
    config_set.add_argument("--from-file", dest="from_file")

    # config list
    config_list = config_sub.add_parser("list")
    config_list.add_argument("--prefix")
    config_list.add_argument("--with-meta", dest="with_meta", action="store_true")

    # config delete
    config_delete = config_sub.add_parser("delete")
    config_delete.add_argument("key")

    # config set-meta
    config_set_meta = config_sub.add_parser("set-meta")
    config_set_meta.add_argument("key")
    config_set_meta.add_argument("metadata", help="JSON metadata")

    # config seed
    config_seed = config_sub.add_parser("seed")
    config_seed.add_argument("--force", action="store_true")
    config_seed.add_argument("--file", default=None,
        help="Path to YAML defaults (defaults to .claude/config-defaults.yaml relative to repo root)")

    return parser


def parse_duration(value: str) -> int:
    """Parse a duration string like '7d' into days."""
    if value.endswith("d"):
        return int(value[:-1])
    return int(value)


def format_entry(entry: dict) -> str:
    """Format an entry for display."""
    date = str(entry["created_at"])[:10]
    status = f"  [{entry['status']}]" if entry.get("status") else ""
    parts = [f"[{date}] {entry['type']}: {entry['title']}{status}"]
    if entry.get("content"):
        parts.append(f"  {entry['content'][:100]}")
    return "\n".join(parts)


def main():
    parser = build_parser()
    args = parser.parse_args()
    use_json = args.json

    db = NeonHTTP.from_env()
    try:
        if args.command == "entry":
            _handle_entry(db, args, use_json)
        elif args.command == "config":
            _handle_config(db, args, use_json)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


def _handle_entry(db, args, use_json: bool):
    if args.action == "add":
        metadata = json.loads(args.metadata) if args.metadata else None
        entry_id = entries.add_entry(
            db, type=args.type, title=args.title, content=args.content,
            status=args.status, metadata=metadata, parent_id=args.parent_id,
        )
        if use_json:
            print(json.dumps({"id": entry_id}))
        else:
            print(f"Created entry {entry_id}")

    elif args.action == "list":
        since_days = parse_duration(args.since) if args.since else None
        due_within_days = parse_duration(args.due_within) if args.due_within else None
        results = entries.list_entries(
            db, type=args.type, status=args.status,
            since_days=since_days, due_within_days=due_within_days,
        )
        if use_json:
            print(json.dumps(results, default=str))
        else:
            for entry in results:
                print(format_entry(entry))

    elif args.action == "search":
        results = entries.search_entries(db, args.query)
        if use_json:
            print(json.dumps(results, default=str))
        else:
            for entry in results:
                print(format_entry(entry))

    elif args.action == "update":
        updates = {}
        if args.title:
            updates["title"] = args.title
        if args.content:
            updates["content"] = args.content
        if args.status:
            updates["status"] = args.status
        if args.parent_id:
            updates["parent_id"] = args.parent_id
        if args.metadata:
            updates["metadata"] = json.loads(args.metadata)
        if entries.update_entry(db, args.id, **updates):
            print(f"Updated entry {args.id}")
        else:
            print(f"Entry {args.id} not found", file=sys.stderr)
            sys.exit(1)

    elif args.action == "delete":
        if entries.delete_entry(db, args.id):
            print(f"Deleted entry {args.id}")
        else:
            print(f"Entry {args.id} not found", file=sys.stderr)
            sys.exit(1)


def _handle_config(db, args, use_json: bool):
    from . import config as config_mod

    if args.action == "get":
        value = config_mod.get(db, args.key)
        if value is None:
            print(f"Key not found: {args.key}", file=sys.stderr)
            sys.exit(1)
        print(json.dumps(value))

    elif args.action == "set":
        if args.from_file:
            with open(args.from_file) as f:
                value = json.load(f)
        else:
            if args.value is None:
                print("Error: must provide either VALUE or --from-file", file=sys.stderr)
                sys.exit(1)
            value = json.loads(args.value)
        config_mod.set(db, args.key, value)
        if use_json:
            print(json.dumps({"key": args.key}))
        else:
            print(f"Set {args.key}")

    elif args.action == "list":
        if args.with_meta:
            results = config_mod.list_with_meta(db, prefix=args.prefix)
            if use_json:
                print(json.dumps(results, default=str))
            else:
                for row in results:
                    desc = row["metadata"].get("description", "").strip().split("\n")[0]
                    if desc:
                        print(f"{row['key']} = {json.dumps(row['value'])}  # {desc}")
                    else:
                        print(f"{row['key']} = {json.dumps(row['value'])}")
        else:
            results = config_mod.list_configs(db, prefix=args.prefix)
            if use_json:
                print(json.dumps(results, default=str))
            else:
                for row in results:
                    print(f"{row['key']} = {json.dumps(row['value'])}")

    elif args.action == "delete":
        if config_mod.delete(db, args.key):
            print(f"Deleted {args.key}")
        else:
            print(f"Key not found: {args.key}", file=sys.stderr)
            sys.exit(1)

    elif args.action == "set-meta":
        metadata = json.loads(args.metadata)
        if not isinstance(metadata, dict):
            print("Error: metadata must be a JSON object", file=sys.stderr)
            sys.exit(1)
        config_mod.set_meta(db, args.key, metadata)
        if use_json:
            print(json.dumps({"key": args.key}))
        else:
            print(f"Set metadata for {args.key}")

    elif args.action == "seed":
        path = args.file or _default_seed_path()
        counts = config_mod.seed(db, path, force=args.force)
        if use_json:
            print(json.dumps(counts))
        else:
            print(
                f"Seeded from {path}: "
                f"{counts['inserted']} inserted, "
                f"{counts['updated']} updated, "
                f"{counts['skipped']} skipped"
            )
