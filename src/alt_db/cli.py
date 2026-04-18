"""CLI entry point for alt-db."""

import argparse
import json
import sys

from .connection import NeonHTTP
from . import entries, nutrition_items, nutrition_logs, nutrition_targets, routines


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
    entry_add.add_argument("--tags")
    entry_add.add_argument("--metadata")

    # entry list
    entry_list = entry_sub.add_parser("list")
    entry_list.add_argument("--type")
    entry_list.add_argument("--status")
    entry_list.add_argument("--since")
    entry_list.add_argument("--due-within", dest="due_within")
    entry_list.add_argument("--tag")

    # entry search
    entry_search = entry_sub.add_parser("search")
    entry_search.add_argument("query")

    # entry update
    entry_update = entry_sub.add_parser("update")
    entry_update.add_argument("id")
    entry_update.add_argument("--title")
    entry_update.add_argument("--content")
    entry_update.add_argument("--status")
    entry_update.add_argument("--tags")
    entry_update.add_argument("--metadata")

    # entry delete
    entry_delete = entry_sub.add_parser("delete")
    entry_delete.add_argument("id")

    # --- routine ---
    routine_parser = subparsers.add_parser("routine")
    routine_sub = routine_parser.add_subparsers(dest="action", required=True)

    # routine complete
    routine_complete = routine_sub.add_parser("complete")
    routine_complete.add_argument("name")
    routine_complete.add_argument("category")
    routine_complete.add_argument("--note")

    # routine baseline
    routine_baseline = routine_sub.add_parser("baseline")
    routine_baseline.add_argument("name")
    routine_baseline.add_argument("category")
    routine_baseline.add_argument("--date", required=True)
    routine_baseline.add_argument("--note")

    # routine last
    routine_last = routine_sub.add_parser("last")
    routine_last.add_argument("name")

    # routine all
    routine_sub.add_parser("all")

    # routine history
    routine_history = routine_sub.add_parser("history")
    routine_history.add_argument("name")

    # routine delete
    routine_delete = routine_sub.add_parser("delete")
    routine_delete.add_argument("id")

    # routine update-note
    routine_update_note = routine_sub.add_parser("update-note")
    routine_update_note.add_argument("id")
    routine_update_note.add_argument("--note", required=True)

    # nutrition-item
    ni_parser = subparsers.add_parser("nutrition-item")
    ni_sub = ni_parser.add_subparsers(dest="action", required=True)

    ni_add = ni_sub.add_parser("add")
    ni_add.add_argument("--name", required=True)
    ni_add.add_argument("--calories", type=float, required=True)
    ni_add.add_argument("--protein", type=float, required=True)
    ni_add.add_argument("--source", default="user_registered")

    ni_sub.add_parser("list")

    ni_get = ni_sub.add_parser("get")
    ni_get.add_argument("--name", required=True)

    ni_update = ni_sub.add_parser("update")
    ni_update.add_argument("--name", required=True)
    ni_update.add_argument("--calories", type=float)
    ni_update.add_argument("--protein", type=float)

    ni_delete = ni_sub.add_parser("delete")
    ni_delete.add_argument("--name", required=True)

    # nutrition-log
    nl_parser = subparsers.add_parser("nutrition-log")
    nl_sub = nl_parser.add_subparsers(dest="action", required=True)

    nl_add = nl_sub.add_parser("add")
    nl_add.add_argument("--date", required=True)
    nl_add.add_argument("--meal-type", required=True)
    nl_add.add_argument("--description", required=True)
    nl_add.add_argument("--calories", type=float)
    nl_add.add_argument("--protein", type=float)
    nl_add.add_argument("--supplement", action="store_true")
    nl_add.add_argument("--source-message-id")
    nl_add.add_argument("--estimated-by", default="llm")

    nl_list = nl_sub.add_parser("list")
    nl_list.add_argument("--date", required=True)

    nl_check = nl_sub.add_parser("check-message")
    nl_check.add_argument("--message-id", required=True)

    nl_summary = nl_sub.add_parser("summary")
    nl_summary.add_argument("--date", required=True)

    # nutrition-target
    nt_parser = subparsers.add_parser("nutrition-target")
    nt_sub = nt_parser.add_subparsers(dest="action", required=True)

    nt_add = nt_sub.add_parser("add")
    nt_add.add_argument("--calories", type=float, required=True)
    nt_add.add_argument("--protein", type=float, required=True)
    nt_add.add_argument("--effective-from", required=True)
    nt_add.add_argument("--rationale")

    nt_sub.add_parser("active")

    nt_sub.add_parser("list")

    return parser


def parse_duration(value: str) -> int:
    """Parse a duration string like '7d' into days."""
    if value.endswith("d"):
        return int(value[:-1])
    return int(value)


def format_entry(entry: dict) -> str:
    """Format an entry for display."""
    date = str(entry["created_at"])[:10]
    tags = " ".join(f"#{t}" for t in entry.get("tags", []))
    status = f"  [{entry['status']}]" if entry.get("status") else ""
    parts = [f"[{date}] {entry['type']}: {entry['title']}{status}"]
    if tags:
        parts[0] += f"  {tags}"
    if entry.get("content"):
        parts.append(f"  {entry['content'][:100]}")
    return "\n".join(parts)


def format_routine_event(event: dict) -> str:
    """Format a routine event for display."""
    date = str(event["completed_at"])[:10]
    note = f"  ({event['note']})" if event.get("note") else ""
    return f"{event['category']:<12} {event['routine_name']:<35} {date}  {event['kind']}{note}"


def main():
    parser = build_parser()
    args = parser.parse_args()
    use_json = args.json

    db = NeonHTTP.from_env()
    try:
        _dispatch(db, args, use_json)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


def _dispatch(db, args, use_json: bool):
    if args.command == "entry":
        _handle_entry(db, args, use_json)
    elif args.command == "routine":
        _handle_routine(db, args, use_json)
    elif args.command == "nutrition-item":
        _handle_nutrition_item(db, args, use_json)
    elif args.command == "nutrition-log":
        _handle_nutrition_log(db, args, use_json)
    elif args.command == "nutrition-target":
        _handle_nutrition_target(db, args, use_json)


def _handle_entry(db, args, use_json: bool):
    if args.action == "add":
        tags = json.loads(args.tags) if args.tags else None
        metadata = json.loads(args.metadata) if args.metadata else None
        entry_id = entries.add_entry(
            db, type=args.type, title=args.title, content=args.content,
            status=args.status, tags=tags, metadata=metadata,
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
            since_days=since_days, due_within_days=due_within_days, tag=args.tag,
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
        if args.tags:
            updates["tags"] = json.loads(args.tags)
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


def _handle_routine(db, args, use_json: bool):
    if args.action == "complete":
        routines.complete_routine(db, args.name, args.category, note=args.note)
        print(f"Marked '{args.name}' as completed")

    elif args.action == "baseline":
        routines.add_baseline(db, args.name, args.category, date=args.date, note=args.note)
        print(f"Set baseline for '{args.name}' at {args.date}")

    elif args.action == "last":
        result = routines.get_last_event(db, args.name)
        if result is None:
            print(f"No events found for '{args.name}'")
        elif use_json:
            print(json.dumps(result, default=str))
        else:
            print(format_routine_event(result))

    elif args.action == "all":
        results = routines.get_all_last_events(db)
        if use_json:
            print(json.dumps(results, default=str))
        else:
            for event in results:
                print(format_routine_event(event))

    elif args.action == "history":
        results = routines.get_history(db, args.name)
        if not results:
            print(f"No events found for '{args.name}'")
        elif use_json:
            print(json.dumps(results, default=str))
        else:
            print(f"{'ID':<38} {'Date':<12} {'Kind':<10} Note")
            for event in results:
                date = str(event["completed_at"])[:10]
                note = event["note"] or "—"
                print(f"{event['id']:<38} {date:<12} {event['kind']:<10} {note}")

    elif args.action == "delete":
        if routines.delete_event(db, args.id):
            print(f"Deleted routine event {args.id}")
        else:
            print(f"Routine event {args.id} not found", file=sys.stderr)
            sys.exit(1)

    elif args.action == "update-note":
        note = args.note if args.note != "" else None
        if routines.update_note(db, args.id, note):
            print(f"Updated note for {args.id}")
        else:
            print(f"Routine event {args.id} not found", file=sys.stderr)
            sys.exit(1)


def _handle_nutrition_item(db, args, use_json: bool):
    if args.action == "add":
        item = nutrition_items.add_item(
            db, name=args.name, calories=args.calories,
            protein=args.protein, source=args.source,
        )
        if use_json:
            print(json.dumps(item))
        else:
            print(f"Added: {item['name']} ({item['calories_kcal']}kcal, P{item['protein_g']}g)")
    elif args.action == "list":
        items = nutrition_items.list_items(db)
        if use_json:
            print(json.dumps(items))
        else:
            for item in items:
                print(f"  {item['name']}: {item['calories_kcal']}kcal, P{item['protein_g']}g [{item['source']}]")
    elif args.action == "get":
        item = nutrition_items.get_item_by_name(db, args.name)
        if item is None:
            print(f"Not found: {args.name}", file=sys.stderr)
            sys.exit(1)
        if use_json:
            print(json.dumps(item))
        else:
            print(f"  {item['name']}: {item['calories_kcal']}kcal, P{item['protein_g']}g [{item['source']}]")
    elif args.action == "update":
        ok = nutrition_items.update_item(db, args.name, calories=args.calories, protein=args.protein)
        if not ok:
            print(f"Not found: {args.name}", file=sys.stderr)
            sys.exit(1)
        print(f"Updated: {args.name}")
    elif args.action == "delete":
        ok = nutrition_items.delete_item(db, args.name)
        if not ok:
            print(f"Not found: {args.name}", file=sys.stderr)
            sys.exit(1)
        print(f"Deleted: {args.name}")


def _handle_nutrition_log(db, args, use_json: bool):
    if args.action == "add":
        log = nutrition_logs.add_log(
            db, logged_date=args.date, meal_type=args.meal_type,
            description=args.description, calories=args.calories,
            protein=args.protein, supplement_taken=args.supplement,
            source_message_id=args.source_message_id, estimated_by=args.estimated_by,
        )
        if use_json:
            print(json.dumps(log))
        else:
            print(f"Logged: {log['description']} ({log['calories_kcal']}kcal, P{log['protein_g']}g)")
    elif args.action == "list":
        logs = nutrition_logs.list_logs_by_date(db, args.date)
        if use_json:
            print(json.dumps(logs))
        else:
            for log in logs:
                print(f"  [{log['meal_type']}] {log['description']}: {log['calories_kcal']}kcal, P{log['protein_g']}g")
    elif args.action == "check-message":
        exists = nutrition_logs.is_message_processed(db, args.message_id)
        if use_json:
            print(json.dumps({"processed": exists}))
        else:
            print("Already processed" if exists else "Not processed")
    elif args.action == "summary":
        summary = nutrition_logs.daily_summary(db, args.date)
        if use_json:
            print(json.dumps(summary))
        else:
            print(f"  Calories: {summary['total_calories']}kcal")
            print(f"  Protein: {summary['total_protein']}g")
            print(f"  Supplement: {'✅' if summary['supplement_taken'] else '❌'}")


def _handle_nutrition_target(db, args, use_json: bool):
    if args.action == "add":
        target = nutrition_targets.add_target(
            db, calories=args.calories, protein=args.protein,
            effective_from=args.effective_from, rationale=args.rationale,
        )
        if use_json:
            print(json.dumps(target))
        else:
            print(f"Target set: {target['calories_kcal']}kcal, P{target['protein_g']}g from {target['effective_from']}")
    elif args.action == "active":
        from datetime import date
        target = nutrition_targets.get_active_target(db, str(date.today()))
        if target is None:
            print("No active target", file=sys.stderr)
            sys.exit(1)
        if use_json:
            print(json.dumps(target))
        else:
            print(f"  Calories: {target['calories_kcal']}kcal")
            print(f"  Protein: {target['protein_g']}g")
            print(f"  Since: {target['effective_from']}")
            if target['rationale']:
                print(f"  Rationale: {target['rationale']}")
    elif args.action == "list":
        targets = nutrition_targets.list_targets(db)
        if use_json:
            print(json.dumps(targets))
        else:
            for t in targets:
                active = "✅" if t["effective_until"] is None else f"→{t['effective_until']}"
                print(f"  {t['effective_from']} {active}: {t['calories_kcal']}kcal, P{t['protein_g']}g")
