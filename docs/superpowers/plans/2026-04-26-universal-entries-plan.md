# Universal Entries Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate 7 database tables into a single universal `entries` table, simplify CLI to entry-only operations, and update all dependent code (webapp, skills, body import).

**Architecture:** Modify the entries table (add parent_id, remove tags), migrate all data from 6 specialized tables into entries as typed rows with JSONB metadata, then update Python CLI, webapp queries, and skill instructions to use the unified table.

**Tech Stack:** PostgreSQL (Neon), Atlas (schema migrations), Python 3.12+ (CLI), Next.js (webapp), Claude Code skills (markdown)

**Design spec:** `docs/superpowers/specs/2026-04-22-universal-entries-design.md`

---

## File Map

### Create
- `db/migrations/YYYYMMDDHHMMSS_universal_entries.sql` — Consolidation migration

### Modify
- `db/schema/entries.hcl` — Add parent_id, remove tags
- `src/alt_db/entries.py` — Remove tags, add parent_id, use explicit column list
- `src/alt_db/cli.py` — Remove specialized subcommands, simplify to entry-only
- `src/alt_body/storage.py` — Insert into entries instead of body_measurements
- `tests/conftest.py` — Simplify cleanup to entries-only
- `tests/test_entries.py` — Update for new schema
- `tests/test_cli.py` — Update for simplified CLI
- `tests/test_body_storage.py` — Update for entries table
- `webapp/src/lib/types.ts` — Remove RoutineEvent, remove tags from Entry, add parent_id
- `webapp/src/lib/queries.ts` — Rewrite getLatestRoutineEvents to query entries
- `webapp/src/lib/body.ts` — Rewrite all queries to use entries + metadata
- `webapp/src/app/routines/page.tsx` — Update to use Entry with metadata
- `.claude/skills/routines/SKILL.md` — Replace `alt-db routine` commands with `alt-db entry`
- `.claude/skills/nutrition-check-cloud/SKILL.md` — Replace nutrition-specific commands with `alt-db entry`
- `README.md` — Add "Why a single table?" section, update architecture
- `CLAUDE.md` — Update routine reference
- `.claude/rules/database-entries.md` — Update CLI examples

### Delete
- `db/schema/routine_events.hcl`
- `db/schema/body_measurements.hcl`
- `db/schema/body_measurement_goals.hcl`
- `db/schema/nutrition_items.hcl`
- `db/schema/nutrition_logs.hcl`
- `db/schema/nutrition_targets.hcl`
- `src/alt_db/routines.py`
- `src/alt_db/nutrition_items.py`
- `src/alt_db/nutrition_logs.py`
- `src/alt_db/nutrition_targets.py`
- `tests/test_routines.py`
- `tests/test_nutrition_items.py`
- `tests/test_nutrition_logs.py`
- `tests/test_nutrition_targets.py`

---

### Task 1: Database Schema and Migration

**Files:**
- Modify: `db/schema/entries.hcl`
- Delete: `db/schema/routine_events.hcl`, `db/schema/body_measurements.hcl`, `db/schema/body_measurement_goals.hcl`, `db/schema/nutrition_items.hcl`, `db/schema/nutrition_logs.hcl`, `db/schema/nutrition_targets.hcl`
- Create: `db/migrations/YYYYMMDDHHMMSS_universal_entries.sql`

- [ ] **Step 1: Run existing tests to confirm baseline**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/ -v --tb=short 2>&1 | tail -20`
Expected: All tests pass (or note any pre-existing failures)

- [ ] **Step 2: Update entries.hcl — add parent_id, remove tags**

Replace the entire content of `db/schema/entries.hcl` with:

```hcl
schema "public" {}

table "entries" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "type" {
    type = text
    null = false
  }
  column "title" {
    type = text
    null = false
  }
  column "content" {
    type = text
    null = true
  }
  column "status" {
    type = text
    null = true
  }
  column "metadata" {
    type    = jsonb
    default = "{}"
  }
  column "parent_id" {
    type = uuid
    null = true
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_entries_parent" {
    columns     = [column.parent_id]
    ref_columns = [column.id]
    on_delete   = SET_NULL
  }

  index "idx_entries_type" {
    columns = [column.type]
  }
  index "idx_entries_created" {
    columns = [column.created_at]
  }
  index "idx_entries_parent" {
    columns = [column.parent_id]
    where   = "parent_id IS NOT NULL"
  }
}
```

- [ ] **Step 3: Delete old schema HCL files**

```bash
rm db/schema/routine_events.hcl
rm db/schema/body_measurements.hcl
rm db/schema/body_measurement_goals.hcl
rm db/schema/nutrition_items.hcl
rm db/schema/nutrition_logs.hcl
rm db/schema/nutrition_targets.hcl
```

Verify only `entries.hcl` remains: `ls db/schema/`
Expected: `entries.hcl`

- [ ] **Step 4: Write the consolidation migration**

Create a new migration file. Use the current timestamp for the filename (format: `YYYYMMDDHHMMSS`). Save to `db/migrations/<timestamp>_universal_entries.sql`:

```sql
-- Universal entries consolidation: merge 6 tables into entries
-- See docs/superpowers/specs/2026-04-22-universal-entries-design.md

-- 1. Add parent_id column
ALTER TABLE entries ADD COLUMN parent_id uuid REFERENCES entries(id) ON DELETE SET NULL;
CREATE INDEX idx_entries_parent ON entries (parent_id) WHERE parent_id IS NOT NULL;

-- 2. Preserve existing tags in metadata before dropping
UPDATE entries
SET metadata = jsonb_set(COALESCE(metadata, '{}'), '{tags}', COALESCE(tags, '[]'))
WHERE tags IS NOT NULL AND tags != '[]'::jsonb;

-- 3. Drop tags column and its GIN index
DROP INDEX IF EXISTS idx_entries_tags;
ALTER TABLE entries DROP COLUMN tags;

-- 4. Migrate routine_events → entries (type='routine_event')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'routine_event',
    routine_name,
    note,
    kind,
    jsonb_build_object(
        'category', category,
        'completed_at', completed_at
    ),
    completed_at,
    completed_at
FROM routine_events;

-- 5. Migrate body_measurements → entries (type='body_measurement')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'body_measurement',
    'InBody ' || to_char(measured_at AT TIME ZONE 'Asia/Tokyo', 'YYYY-MM-DD'),
    NULL,
    NULL,
    jsonb_build_object(
        'measured_at', measured_at,
        'weight_kg', weight_kg,
        'skeletal_muscle_mass_kg', skeletal_muscle_mass_kg,
        'muscle_mass_kg', muscle_mass_kg,
        'body_fat_mass_kg', body_fat_mass_kg,
        'body_fat_percent', body_fat_percent,
        'bmi', bmi,
        'basal_metabolic_rate', basal_metabolic_rate,
        'inbody_score', inbody_score,
        'waist_hip_ratio', waist_hip_ratio,
        'visceral_fat_level', visceral_fat_level,
        'ffmi', ffmi,
        'skeletal_muscle_ratio', skeletal_muscle_ratio,
        'source', source
    ),
    created_at,
    created_at
FROM body_measurements;

-- 6. Migrate body_measurement_goals → entries (type='body_measurement_goal')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'body_measurement_goal',
    metric,
    NULL,
    status,
    jsonb_build_object(
        'metric', metric,
        'target_value', target_value,
        'start_value', start_value,
        'start_date', start_date,
        'target_date', target_date
    ),
    created_at,
    created_at
FROM body_measurement_goals;

-- 7. Migrate nutrition_items → entries (type='nutrition_item')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'nutrition_item',
    name,
    NULL,
    NULL,
    jsonb_build_object(
        'calories_kcal', calories_kcal,
        'protein_g', protein_g,
        'source', source
    ),
    created_at,
    updated_at
FROM nutrition_items;

-- 8. Migrate nutrition_logs → entries (type='nutrition_log')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'nutrition_log',
    description,
    NULL,
    NULL,
    jsonb_build_object(
        'logged_date', logged_date,
        'meal_type', meal_type,
        'calories_kcal', calories_kcal,
        'protein_g', protein_g,
        'supplement_taken', supplement_taken,
        'source_message_id', source_message_id,
        'estimated_by', estimated_by
    ),
    created_at,
    updated_at
FROM nutrition_logs;

-- 9. Migrate nutrition_targets → entries (type='nutrition_target')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'nutrition_target',
    'Nutrition Target ' || effective_from,
    rationale,
    CASE WHEN effective_until IS NULL THEN 'active' ELSE 'inactive' END,
    jsonb_build_object(
        'calories_kcal', calories_kcal,
        'protein_g', protein_g,
        'effective_from', effective_from,
        'effective_until', effective_until
    ),
    created_at,
    updated_at
FROM nutrition_targets;

-- 10. Drop old tables
DROP TABLE routine_events;
DROP TABLE body_measurement_goals;
DROP TABLE body_measurements;
DROP TABLE nutrition_logs;
DROP TABLE nutrition_items;
DROP TABLE nutrition_targets;
```

- [ ] **Step 5: Update atlas migration hash**

Run: `cd /Users/makoto/projects/alt/db && atlas migrate hash`
Expected: `atlas.sum` file is updated

- [ ] **Step 6: Apply migration to Neon**

Run: `cd /Users/makoto/projects/alt/db && atlas migrate apply --env neon`
Expected: Migration applied successfully. Verify with: `atlas migrate status --env neon`

- [ ] **Step 7: Commit**

```bash
git add db/schema/entries.hcl db/migrations/ db/atlas.sum
git add -u db/schema/  # stages deletions of old HCL files
git commit -m "feat(db): consolidate 7 tables into universal entries

Add parent_id column, remove tags column, migrate all data from
routine_events, body_measurements, body_measurement_goals,
nutrition_items, nutrition_logs, and nutrition_targets into entries."
```

---

### Task 2: Python — Update entries.py

**Files:**
- Modify: `src/alt_db/entries.py`
- Test: `tests/test_entries.py`

This task uses explicit column lists instead of `SELECT *` to avoid column-order issues after ALTER TABLE.

- [ ] **Step 1: Write updated tests**

Read `tests/test_entries.py` first. Then update it to remove all `tags` references and add `parent_id` coverage. Key changes:

1. Remove `tags` from `add_entry()` calls and assertions
2. Remove the `test_list_entries_by_tag` test
3. Remove `tags` from `test_update_entry_title_and_tags` — change to `test_update_entry_title_and_metadata`
4. Add `parent_id` assertions to `test_add_and_get_entry`
5. Add a test for `parent_id` creation and retrieval

Every test that creates an entry and checks the returned dict: remove `"tags"` key assertions, add `"parent_id": None` assertions.

For the new parent_id test, add after the existing tests:

```python
def test_add_entry_with_parent(db):
    client, created_ids, *_ = db
    parent_id = entries.add_entry(client, type="goal", title="Parent goal")
    created_ids.append(parent_id)
    child_id = entries.add_entry(client, type="goal", title="Sub-goal", parent_id=parent_id)
    created_ids.append(child_id)
    child = entries.get_entry(client, child_id)
    assert child["parent_id"] == parent_id
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/test_entries.py -v --tb=short`
Expected: FAIL — `add_entry()` doesn't accept `parent_id`, dict has no `parent_id` key

- [ ] **Step 3: Update entries.py**

Replace the full content of `src/alt_db/entries.py`:

```python
"""Entry CRUD operations."""

import json

from .connection import NeonHTTP

_COLUMNS = "id, type, title, content, status, metadata, parent_id, created_at, updated_at"


def _next_param(params: list, value) -> str:
    """Append value to params list and return $N placeholder."""
    params.append(value)
    return f"${len(params)}"


def add_entry(
    db: NeonHTTP,
    type: str,
    title: str,
    content: str | None = None,
    status: str | None = None,
    metadata: dict | None = None,
    parent_id: str | None = None,
) -> str:
    """Create a new entry. Returns the entry ID."""
    result = db.execute(
        """
        INSERT INTO entries (type, title, content, status, metadata, parent_id)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
        """,
        [type, title, content, status, json.dumps(metadata or {}), parent_id],
    )
    return str(result.rows[0][0])


def get_entry(db: NeonHTTP, entry_id: str) -> dict | None:
    """Get a single entry by ID."""
    result = db.execute(
        f"SELECT {_COLUMNS} FROM entries WHERE id = $1",
        [entry_id],
    )
    if not result.rows:
        return None
    return _row_to_dict(result.rows[0])


def list_entries(
    db: NeonHTTP,
    type: str | None = None,
    status: str | None = None,
    since_days: int | None = None,
    due_within_days: int | None = None,
) -> list[dict]:
    """List entries with optional filters."""
    conditions = []
    params: list = []

    if type is not None:
        conditions.append(f"type = {_next_param(params, type)}")
    if status is not None:
        conditions.append(f"status = {_next_param(params, status)}")
    if since_days is not None:
        conditions.append(f"created_at >= now() - make_interval(days => {_next_param(params, since_days)})")
    if due_within_days is not None:
        conditions.append(f"(metadata->>'target_date')::date <= (current_date + make_interval(days => {_next_param(params, due_within_days)}))")
        conditions.append("(metadata->>'target_date') IS NOT NULL")

    where = " AND ".join(conditions) if conditions else "TRUE"
    result = db.execute(
        f"SELECT {_COLUMNS} FROM entries WHERE {where} ORDER BY created_at DESC",
        params,
    )
    return [_row_to_dict(row) for row in result.rows]


def search_entries(db: NeonHTTP, query: str) -> list[dict]:
    """Search entries by title and content (case-insensitive)."""
    escaped = query.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_")
    pattern = f"%{escaped}%"
    result = db.execute(
        f"SELECT {_COLUMNS} FROM entries WHERE title ILIKE $1 OR content ILIKE $2 ORDER BY created_at DESC",
        [pattern, pattern],
    )
    return [_row_to_dict(row) for row in result.rows]


def update_entry(db: NeonHTTP, entry_id: str, **updates) -> bool:
    """Update an entry. Supported fields: title, content, status, metadata, parent_id. Returns True if updated."""
    allowed = {"title", "content", "status", "metadata", "parent_id"}
    fields = {k: v for k, v in updates.items() if k in allowed}
    if not fields:
        return False

    set_clauses = []
    params: list = []
    for key, value in fields.items():
        if key == "metadata":
            set_clauses.append(f"{key} = {_next_param(params, json.dumps(value))}")
        else:
            set_clauses.append(f"{key} = {_next_param(params, value)}")

    set_clauses.append("updated_at = now()")

    result = db.execute(
        f"UPDATE entries SET {', '.join(set_clauses)} WHERE id = {_next_param(params, entry_id)}",
        params,
    )
    return result.row_count > 0


def delete_entry(db: NeonHTTP, entry_id: str) -> bool:
    """Delete an entry. Returns True if deleted."""
    result = db.execute("DELETE FROM entries WHERE id = $1", [entry_id])
    return result.row_count > 0


def _row_to_dict(row) -> dict:
    """Convert a database row to a dict. Column order matches _COLUMNS."""
    metadata = row[5]
    if isinstance(metadata, str):
        metadata = json.loads(metadata)
    return {
        "id": str(row[0]),
        "type": row[1],
        "title": row[2],
        "content": row[3],
        "status": row[4],
        "metadata": metadata,
        "parent_id": str(row[6]) if row[6] else None,
        "created_at": str(row[7]),
        "updated_at": str(row[8]),
    }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/test_entries.py -v --tb=short`
Expected: All tests pass

Note: Task 1's migration must be applied to Neon first.

- [ ] **Step 5: Commit**

```bash
git add src/alt_db/entries.py tests/test_entries.py
git commit -m "refactor(alt-db): update entries module for universal schema

Remove tags support, add parent_id, use explicit column list."
```

---

### Task 3: Python — Simplify CLI to Entry-Only

**Files:**
- Modify: `src/alt_db/cli.py`
- Test: `tests/test_cli.py`

- [ ] **Step 1: Update test_cli.py**

Read `tests/test_cli.py`. Key changes:
1. Remove all tests for `routine` subcommand argument parsing
2. Remove all tests for `nutrition-item`, `nutrition-log`, `nutrition-target` argument parsing
3. Remove tests for `format_routine_event`
4. Update `test_entry_add_with_tags` → remove `--tags` arg, add `--parent-id` test
5. Update `test_entry_list_args` → remove `--tag` arg assertion
6. Update `test_entry_update_args` → remove `--tags` arg, add `--parent-id`
7. Update `test_format_entry` → remove tags from expected output

For the format_entry test, the expected output changes since tags are removed:

```python
def test_format_entry():
    entry = {
        "created_at": "2025-01-15T10:00:00+00:00",
        "type": "memo",
        "title": "Test note",
        "status": "active",
        "content": "Some content here",
    }
    result = format_entry(entry)
    assert "[2025-01-15]" in result
    assert "memo: Test note" in result
    assert "[active]" in result
    assert "Some content here" in result
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/test_cli.py -v --tb=short`
Expected: FAIL — old subcommands still exist, --tags still accepted

- [ ] **Step 3: Rewrite cli.py**

Replace the full content of `src/alt_db/cli.py`:

```python
"""CLI entry point for alt-db."""

import argparse
import json
import sys

from .connection import NeonHTTP
from . import entries


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
    entry_add.add_argument("--metadata")
    entry_add.add_argument("--parent-id", dest="parent_id")

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
    entry_update.add_argument("--metadata")
    entry_update.add_argument("--parent-id", dest="parent_id")

    # entry delete
    entry_delete = entry_sub.add_parser("delete")
    entry_delete.add_argument("id")

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
        _handle_entry(db, args, use_json)
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
        if args.metadata:
            updates["metadata"] = json.loads(args.metadata)
        if args.parent_id:
            updates["parent_id"] = args.parent_id
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/test_cli.py -v --tb=short`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add src/alt_db/cli.py tests/test_cli.py
git commit -m "refactor(alt-db): simplify CLI to entry-only operations

Remove routine, nutrition-item, nutrition-log, nutrition-target
subcommands. All operations now go through 'alt-db entry'."
```

---

### Task 4: Python — Update Body Storage

**Files:**
- Modify: `src/alt_body/storage.py`
- Test: `tests/test_body_storage.py`

- [ ] **Step 1: Update test_body_storage.py**

Read `tests/test_body_storage.py`. Update to verify that `upsert_measurements` inserts into entries table with type='body_measurement'. Key changes:

1. The test fixture cleanup uses `created_entry_ids` instead of `created_body_timestamps`
2. After insert, verify by querying entries table: `SELECT metadata FROM entries WHERE type = 'body_measurement' AND metadata->>'measured_at' = $1`
3. Duplicate detection is now application-level (check before insert)

For the dedup test, insert the same measurement twice and verify `(1, 1)` is returned (1 inserted, 1 skipped). Verify only one entry exists.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/test_body_storage.py -v --tb=short`
Expected: FAIL — storage.py still references body_measurements table

- [ ] **Step 3: Rewrite storage.py**

Replace the full content of `src/alt_body/storage.py`:

```python
"""Store body measurements in Neon Postgres via the entries table."""

import json

from alt_db.connection import NeonHTTP

_MEASUREMENT_FIELDS = [
    "measured_at",
    "weight_kg",
    "skeletal_muscle_mass_kg",
    "muscle_mass_kg",
    "body_fat_mass_kg",
    "body_fat_percent",
    "bmi",
    "basal_metabolic_rate",
    "inbody_score",
    "waist_hip_ratio",
    "visceral_fat_level",
    "ffmi",
    "skeletal_muscle_ratio",
]


def upsert_measurements(
    db: NeonHTTP, measurements: list[dict]
) -> tuple[int, int]:
    """Insert measurements as entries, skipping duplicates. Returns (inserted, skipped)."""
    inserted = 0
    skipped = 0

    for m in measurements:
        measured_at = m["measured_at"]
        if hasattr(measured_at, "isoformat"):
            measured_at = measured_at.isoformat()

        # Check for existing entry with same measured_at
        existing = db.execute(
            "SELECT id FROM entries WHERE type = 'body_measurement' AND metadata->>'measured_at' = $1",
            [measured_at],
        )
        if existing.rows:
            skipped += 1
            continue

        # Build metadata from measurement fields
        metadata = {}
        for field in _MEASUREMENT_FIELDS:
            value = m[field]
            if hasattr(value, "isoformat"):
                value = value.isoformat()
            metadata[field] = value

        title = "InBody " + str(measured_at)[:10]

        db.execute(
            "INSERT INTO entries (type, title, metadata) VALUES ($1, $2, $3)",
            ["body_measurement", title, json.dumps(metadata)],
        )
        inserted += 1

    return inserted, skipped
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/test_body_storage.py -v --tb=short`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add src/alt_body/storage.py tests/test_body_storage.py
git commit -m "refactor(alt-body): store measurements in entries table

Use type='body_measurement' with all InBody fields in metadata."
```

---

### Task 5: Delete Old Modules and Update Conftest

**Files:**
- Delete: `src/alt_db/routines.py`, `src/alt_db/nutrition_items.py`, `src/alt_db/nutrition_logs.py`, `src/alt_db/nutrition_targets.py`
- Delete: `tests/test_routines.py`, `tests/test_nutrition_items.py`, `tests/test_nutrition_logs.py`, `tests/test_nutrition_targets.py`
- Modify: `tests/conftest.py`

- [ ] **Step 1: Delete old Python modules**

```bash
rm src/alt_db/routines.py
rm src/alt_db/nutrition_items.py
rm src/alt_db/nutrition_logs.py
rm src/alt_db/nutrition_targets.py
```

- [ ] **Step 2: Delete old test files**

```bash
rm tests/test_routines.py
rm tests/test_nutrition_items.py
rm tests/test_nutrition_logs.py
rm tests/test_nutrition_targets.py
```

- [ ] **Step 3: Simplify conftest.py**

Replace the full content of `tests/conftest.py`:

```python
"""Shared test fixtures."""

import pytest

from alt_db.connection import NeonHTTP


@pytest.fixture
def db():
    """Provide a NeonHTTP client with DELETE-based cleanup after each test."""
    client = NeonHTTP.from_env()
    created_entry_ids: list[str] = []
    yield client, created_entry_ids
    # Teardown: delete test entries
    for eid in created_entry_ids:
        client.execute("DELETE FROM entries WHERE id = $1", [eid])
```

- [ ] **Step 4: Update test files that use the fixture**

`test_entries.py` and `test_body_storage.py` currently unpack the fixture as `client, created_ids, *_`. After conftest changes, the fixture yields `(client, created_entry_ids)`. Update any test that unpacks more than 2 values:

```python
# Before:
client, created_ids, *_ = db
# After:
client, created_ids = db
```

- [ ] **Step 5: Run all tests**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/ -v --tb=short`
Expected: All remaining tests pass. No import errors from deleted modules.

- [ ] **Step 6: Commit**

```bash
git add -u src/alt_db/ tests/
git add tests/conftest.py
git commit -m "chore(alt-db): remove specialized modules and simplify test fixtures

Delete routines, nutrition_items, nutrition_logs, nutrition_targets
modules and their tests. Simplify conftest to entries-only cleanup."
```

---

### Task 6: Webapp — Types, Queries, and Routines Page

**Files:**
- Modify: `webapp/src/lib/types.ts`
- Modify: `webapp/src/lib/queries.ts`
- Modify: `webapp/src/app/routines/page.tsx`

- [ ] **Step 1: Update types.ts**

Read `webapp/src/lib/types.ts`. Make these changes:

1. Remove `tags` field from `Entry` interface, add `parent_id`
2. Remove `RoutineEvent` interface entirely
3. Remove `tag` field from `EntryFilters` interface

New content of `webapp/src/lib/types.ts`:

```typescript
export interface Entry {
  id: string
  type: string
  title: string
  content: string | null
  status: string | null
  metadata: Record<string, unknown>
  parent_id: string | null
  created_at: string
  updated_at: string
}

export interface EntryFilters {
  type?: string | null
  status?: string | null
  search?: string | null
  limit?: number
}

export interface BodyMeasurement {
  id: string
  measured_at: string
  weight_kg: number
  skeletal_muscle_mass_kg: number | null
  muscle_mass_kg: number | null
  body_fat_mass_kg: number | null
  body_fat_percent: number | null
  bmi: number | null
  basal_metabolic_rate: number | null
  inbody_score: number | null
  waist_hip_ratio: number | null
  visceral_fat_level: number | null
  ffmi: number | null
  skeletal_muscle_ratio: number | null
  source: string
  created_at: string
}

export type BodyGoalStatus = "active" | "achieved" | "expired" | "cancelled"

export interface BodyGoal {
  id: string
  metric: string
  target_value: number
  start_value: number | null
  start_date: string
  target_date: string
  status: BodyGoalStatus
  created_at: string
}

export type Period = "30d" | "90d" | "6m" | "1y" | "all"
```

- [ ] **Step 2: Update queries.ts — rewrite getLatestRoutineEvents and listEntries**

Read `webapp/src/lib/queries.ts`. Make these changes:

1. Remove `RoutineEvent` import
2. Rename `getLatestRoutineEvents()` → `getLatestRoutineEntries()`, query entries table
3. Remove `tag` filter from `listEntries()`
4. Use explicit column list in all SELECT queries (replace `SELECT *` with column names including `parent_id`)

Replace `getLatestRoutineEvents`:

```typescript
export async function getLatestRoutineEntries(): Promise<Entry[]> {
  const rows = await sql`
    SELECT DISTINCT ON (title) id, type, title, content, status, metadata, parent_id, created_at, updated_at
    FROM entries
    WHERE type = 'routine_event'
    ORDER BY title, created_at DESC
  `
  return rows as Entry[]
}
```

Update `listEntries` — remove the tag filter and add `parent_id` to SELECT:

```typescript
export async function listEntries(filters: EntryFilters): Promise<Entry[]> {
  const { type = null, status = null, search = null, limit = 50 } = filters
  const searchPattern = search ? `%${search}%` : null

  const rows = await sql`
    SELECT id, type, title, content, status, metadata, parent_id, created_at, updated_at
    FROM entries
    WHERE (${type}::text IS NULL OR type = ${type})
      AND (${status}::text IS NULL OR status = ${status})
      AND (${searchPattern}::text IS NULL OR title ILIKE ${searchPattern} OR content ILIKE ${searchPattern})
    ORDER BY created_at DESC
    LIMIT ${limit}
  `
  return rows as Entry[]
}
```

Also update all other query functions (`getActiveGoals`, `getRecentMemos`, `getUpcomingDeadlines`, `getXDrafts`, `getOccupiedSlots`) — change `SELECT *` to explicit column list including `parent_id`.

- [ ] **Step 3: Update routines page.tsx**

Replace the full content of `webapp/src/app/routines/page.tsx`:

```tsx
import { getLatestRoutineEntries } from "@/lib/queries"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"

export default async function RoutinesPage() {
  const events = await getLatestRoutineEntries()

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="text-3xl font-bold mb-6">Routines</h1>

      {events.length === 0 ? (
        <p className="text-muted-foreground">No routine events recorded.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Routine</TableHead>
              <TableHead className="w-32">Category</TableHead>
              <TableHead className="w-32">Kind</TableHead>
              <TableHead className="w-40">Last Completed</TableHead>
              <TableHead>Note</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.map((event) => (
              <TableRow key={event.id}>
                <TableCell className="font-medium">{event.title}</TableCell>
                <TableCell>
                  <Badge variant="outline">{String(event.metadata.category ?? "")}</Badge>
                </TableCell>
                <TableCell>
                  <Badge variant="secondary">{event.status ?? ""}</Badge>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {event.metadata.completed_at
                    ? new Date(String(event.metadata.completed_at)).toLocaleDateString()
                    : "—"}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {event.content || "—"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
```

- [ ] **Step 4: Verify webapp builds**

Run: `cd /Users/makoto/projects/alt/webapp && npm run build 2>&1 | tail -20`
Expected: Build succeeds with no type errors

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/types.ts webapp/src/lib/queries.ts webapp/src/app/routines/page.tsx
git commit -m "refactor(webapp): update types, queries, and routines page for universal entries

Remove RoutineEvent type, remove tags from Entry, query entries
table for routine events."
```

---

### Task 7: Webapp — Body Module

**Files:**
- Modify: `webapp/src/lib/body.ts`

Strategy: Transform entries with type='body_measurement' back into BodyMeasurement shape in the query layer, so pages, components, and API routes don't need to change.

- [ ] **Step 1: Rewrite body.ts**

Read `webapp/src/lib/body.ts`. Replace the full content with the entries-based version. Key pattern: query entries, extract metadata into the expected typed shape via `entryToMeasurement()` and `entryToGoal()` helper functions.

```typescript
import { sql } from "./db"
import type { BodyMeasurement, BodyGoal, BodyGoalStatus } from "./types"

function serializeDateOnly(value: unknown): string {
  if (value instanceof Date) {
    const y = value.getFullYear()
    const m = String(value.getMonth() + 1).padStart(2, "0")
    const d = String(value.getDate()).padStart(2, "0")
    return `${y}-${m}-${d}`
  }
  return String(value ?? "")
}

function entryToMeasurement(row: Record<string, unknown>): BodyMeasurement {
  const meta = (row.metadata ?? {}) as Record<string, unknown>
  return {
    id: row.id as string,
    measured_at: String(meta.measured_at ?? ""),
    weight_kg: Number(meta.weight_kg),
    skeletal_muscle_mass_kg: meta.skeletal_muscle_mass_kg != null ? Number(meta.skeletal_muscle_mass_kg) : null,
    muscle_mass_kg: meta.muscle_mass_kg != null ? Number(meta.muscle_mass_kg) : null,
    body_fat_mass_kg: meta.body_fat_mass_kg != null ? Number(meta.body_fat_mass_kg) : null,
    body_fat_percent: meta.body_fat_percent != null ? Number(meta.body_fat_percent) : null,
    bmi: meta.bmi != null ? Number(meta.bmi) : null,
    basal_metabolic_rate: meta.basal_metabolic_rate != null ? Number(meta.basal_metabolic_rate) : null,
    inbody_score: meta.inbody_score != null ? Number(meta.inbody_score) : null,
    waist_hip_ratio: meta.waist_hip_ratio != null ? Number(meta.waist_hip_ratio) : null,
    visceral_fat_level: meta.visceral_fat_level != null ? Number(meta.visceral_fat_level) : null,
    ffmi: meta.ffmi != null ? Number(meta.ffmi) : null,
    skeletal_muscle_ratio: meta.skeletal_muscle_ratio != null ? Number(meta.skeletal_muscle_ratio) : null,
    source: String(meta.source ?? "inbody_csv"),
    created_at: String(row.created_at ?? ""),
  }
}

function entryToGoal(row: Record<string, unknown>): BodyGoal {
  const meta = (row.metadata ?? {}) as Record<string, unknown>
  return {
    id: row.id as string,
    metric: String(meta.metric ?? row.title ?? ""),
    target_value: Number(meta.target_value),
    start_value: meta.start_value != null ? Number(meta.start_value) : null,
    start_date: serializeDateOnly(meta.start_date),
    target_date: serializeDateOnly(meta.target_date),
    status: (row.status ?? "active") as BodyGoalStatus,
    created_at: String(row.created_at ?? ""),
  }
}

export async function getBodyMeasurements(
  startDate: string | null
): Promise<BodyMeasurement[]> {
  if (startDate) {
    const rows = await sql`
      SELECT id, metadata, created_at FROM entries
      WHERE type = 'body_measurement'
        AND (metadata->>'measured_at')::timestamptz >= ${startDate}::date
      ORDER BY (metadata->>'measured_at')::timestamptz ASC
    `
    return rows.map((r) => entryToMeasurement(r as Record<string, unknown>))
  }
  const rows = await sql`
    SELECT id, metadata, created_at FROM entries
    WHERE type = 'body_measurement'
    ORDER BY (metadata->>'measured_at')::timestamptz ASC
  `
  return rows.map((r) => entryToMeasurement(r as Record<string, unknown>))
}

export async function getLatestMeasurement(): Promise<BodyMeasurement | null> {
  const rows = await sql`
    SELECT id, metadata, created_at FROM entries
    WHERE type = 'body_measurement'
    ORDER BY (metadata->>'measured_at')::timestamptz DESC
    LIMIT 1
  `
  if (rows.length === 0) return null
  return entryToMeasurement(rows[0] as Record<string, unknown>)
}

export async function getBodyGoals(
  status: BodyGoalStatus | null = null
): Promise<BodyGoal[]> {
  if (status) {
    const rows = await sql`
      SELECT id, title, status, metadata, created_at FROM entries
      WHERE type = 'body_measurement_goal' AND status = ${status}
      ORDER BY created_at DESC
    `
    return rows.map((r) => entryToGoal(r as Record<string, unknown>))
  }
  const rows = await sql`
    SELECT id, title, status, metadata, created_at FROM entries
    WHERE type = 'body_measurement_goal'
    ORDER BY created_at DESC
  `
  return rows.map((r) => entryToGoal(r as Record<string, unknown>))
}

export async function createBodyGoal(input: {
  metric: string
  target_value: number
  target_date: string
}): Promise<BodyGoal> {
  const latest = await getLatestMeasurement()
  const startValue = latest
    ? (latest[input.metric as keyof BodyMeasurement] as number | null)
    : null
  const startDate = new Date().toISOString().slice(0, 10)

  const metadata = {
    metric: input.metric,
    target_value: input.target_value,
    start_value: startValue,
    start_date: startDate,
    target_date: input.target_date,
  }

  const rows = await sql`
    INSERT INTO entries (type, title, status, metadata)
    VALUES ('body_measurement_goal', ${input.metric}, 'active', ${JSON.stringify(metadata)}::jsonb)
    RETURNING id, title, status, metadata, created_at
  `
  return entryToGoal(rows[0] as Record<string, unknown>)
}

export async function updateBodyGoal(
  id: string,
  input: { target_value?: number; target_date?: string; status?: BodyGoalStatus }
): Promise<BodyGoal> {
  const metaUpdates: Record<string, unknown> = {}
  if (input.target_value !== undefined) metaUpdates.target_value = input.target_value
  if (input.target_date !== undefined) metaUpdates.target_date = input.target_date

  const statusUpdate = input.status ?? null

  if (Object.keys(metaUpdates).length > 0 && statusUpdate) {
    const rows = await sql`
      UPDATE entries
      SET metadata = metadata || ${JSON.stringify(metaUpdates)}::jsonb,
          status = ${statusUpdate},
          updated_at = now()
      WHERE id = ${id}
      RETURNING id, title, status, metadata, created_at
    `
    return entryToGoal(rows[0] as Record<string, unknown>)
  } else if (Object.keys(metaUpdates).length > 0) {
    const rows = await sql`
      UPDATE entries
      SET metadata = metadata || ${JSON.stringify(metaUpdates)}::jsonb,
          updated_at = now()
      WHERE id = ${id}
      RETURNING id, title, status, metadata, created_at
    `
    return entryToGoal(rows[0] as Record<string, unknown>)
  } else {
    const rows = await sql`
      UPDATE entries
      SET status = COALESCE(${statusUpdate}, status),
          updated_at = now()
      WHERE id = ${id}
      RETURNING id, title, status, metadata, created_at
    `
    return entryToGoal(rows[0] as Record<string, unknown>)
  }
}

export async function getAllMeasurementsForCalibration(): Promise<
  { weight_kg: number; muscle_mass_kg: number | null; skeletal_muscle_mass_kg: number | null; body_fat_mass_kg: number | null }[]
> {
  const rows = await sql`
    SELECT metadata FROM entries
    WHERE type = 'body_measurement'
    ORDER BY (metadata->>'measured_at')::timestamptz ASC
  `
  return rows.map((r) => {
    const meta = (r as Record<string, unknown>).metadata as Record<string, unknown>
    return {
      weight_kg: Number(meta.weight_kg),
      muscle_mass_kg: meta.muscle_mass_kg != null ? Number(meta.muscle_mass_kg) : null,
      skeletal_muscle_mass_kg: meta.skeletal_muscle_mass_kg != null ? Number(meta.skeletal_muscle_mass_kg) : null,
      body_fat_mass_kg: meta.body_fat_mass_kg != null ? Number(meta.body_fat_mass_kg) : null,
    }
  })
}

export async function getMeasurementHistory(
  limit: number = 20,
  offset: number = 0
): Promise<{ rows: BodyMeasurement[]; total: number }> {
  const countResult = await sql`
    SELECT COUNT(*)::int as total FROM entries WHERE type = 'body_measurement'
  `
  const rows = await sql`
    SELECT id, metadata, created_at FROM entries
    WHERE type = 'body_measurement'
    ORDER BY (metadata->>'measured_at')::timestamptz DESC
    LIMIT ${limit} OFFSET ${offset}
  `
  return {
    rows: rows.map((r) => entryToMeasurement(r as Record<string, unknown>)),
    total: (countResult[0] as { total: number }).total,
  }
}
```

- [ ] **Step 2: Verify API routes and body page need no changes**

Read `webapp/src/app/api/body/measurements/route.ts`, `webapp/src/app/api/body/goals/route.ts`, `webapp/src/app/api/body/goals/[id]/route.ts`, and `webapp/src/app/body/page.tsx`. These all call functions from `body.ts` using `BodyMeasurement`/`BodyGoal` types. Since body.ts transforms entries back into these types, these files should work without modification.

Verify: no direct SQL queries or table references in these files.

- [ ] **Step 3: Verify webapp builds**

Run: `cd /Users/makoto/projects/alt/webapp && npm run build 2>&1 | tail -20`
Expected: Build succeeds with no type errors

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/body.ts
git commit -m "refactor(webapp): rewrite body module to query entries table

Transform entries with type='body_measurement' and 'body_measurement_goal'
back into BodyMeasurement and BodyGoal types in the query layer.
Pages and API routes need no changes."
```

---

### Task 8: Update Skills

**Files:**
- Modify: `.claude/skills/routines/SKILL.md`
- Modify: `.claude/skills/nutrition-check-cloud/SKILL.md`
- Check: all other skills for `--tags` usage or old subcommand references

- [ ] **Step 1: Update routines skill**

Read `.claude/skills/routines/SKILL.md`. Replace all `alt-db routine` commands with `alt-db entry` equivalents:

| Old command | New command |
|---|---|
| `alt-db routine all` | `alt-db --json entry list --type routine_event` then deduplicate by title (keep latest per routine) |
| `alt-db routine complete "<name>" "<category>" --note "<note>"` | `alt-db entry add --type routine_event --title "<name>" --status completed --content "<note>" --metadata '{"category":"<category>","completed_at":"<ISO timestamp>"}'` |
| `alt-db routine baseline "<name>" "<category>" --date "<date>" --note "<note>"` | `alt-db entry add --type routine_event --title "<name>" --status baseline --content "<note>" --metadata '{"category":"<category>","completed_at":"<date>T00:00:00+09:00"}'` |
| `alt-db routine history "<name>"` | `alt-db --json entry search "<name>"` then filter by type=routine_event |
| `alt-db routine delete <id>` | `alt-db entry delete <id>` |
| `alt-db routine update-note <id> --note "..."` | `alt-db entry update <id> --content "..."` |

Also update the skill's data interpretation: when reading `--json` output, fields are `title` (was `routine_name`), `status` (was `kind`), `content` (was `note`), and `metadata.category`/`metadata.completed_at`.

For the "get last event per routine" pattern: the skill should query all routine_event entries with `--json`, then process the JSON to find the latest entry per `title` by `metadata.completed_at`.

- [ ] **Step 2: Update nutrition-check-cloud skill**

Read `.claude/skills/nutrition-check-cloud/SKILL.md`. Replace all nutrition-specific commands:

| Old command | New command |
|---|---|
| `alt-db nutrition-log add --date <d> --meal-type <mt> --description "<desc>" --calories <c> --protein <p> --source-message-id <id> --estimated-by <method>` | `alt-db entry add --type nutrition_log --title "<desc>" --metadata '{"logged_date":"<d>","meal_type":"<mt>","calories_kcal":<c>,"protein_g":<p>,"source_message_id":"<id>","estimated_by":"<method>"}'` |
| `alt-db --json nutrition-log summary --date <d>` | `alt-db --json entry list --type nutrition_log --since 2d` then filter by `metadata.logged_date == <d>` and compute sums |
| `alt-db --json nutrition-log list --date <d>` | `alt-db --json entry list --type nutrition_log --since 2d` then filter by `metadata.logged_date == <d>` |
| `alt-db --json nutrition-log check-message --message-id <id>` | `alt-db --json entry search "<id>"` then check if any result has type=nutrition_log |
| `alt-db --json nutrition-target active` | `alt-db --json entry list --type nutrition_target --status active` |
| `alt-db --json nutrition-item list` | `alt-db --json entry list --type nutrition_item` |
| `alt-db nutrition-item add --name "<name>" --calories <c> --protein <p>` | `alt-db entry add --type nutrition_item --title "<name>" --metadata '{"calories_kcal":<c>,"protein_g":<p>,"source":"user_registered"}'` |
| `alt-db nutrition-item update --name "<name>" --calories <c>` | Find entry by search, then `alt-db entry update <id> --metadata '...'` |
| `alt-db nutrition-item delete --name "<name>"` | Find entry by search, then `alt-db entry delete <id>` |

- [ ] **Step 3: Verify other skills — remove --tags usage**

Read these skills and update any `--tags` arguments to put the values in metadata instead (or remove them if redundant with the `type` field):
- `.claude/skills/daily-plan-cloud/SKILL.md` — uses `--tags '["daily-plan"]'` → remove (redundant with type=daily_plan)
- `.claude/skills/daily-plan/SKILL.md` — same
- `.claude/skills/weekly-plan-cloud/SKILL.md` — uses `--tags '["weekly-plan"]'` → remove
- `.claude/skills/weekly-plan/SKILL.md` — same
- `.claude/skills/x-draft-cloud/SKILL.md` — uses `--tags '["<project>"]'` → move to metadata
- `.claude/skills/x-post-cloud/SKILL.md` — verify no --tags
- `.claude/skills/wake-check-cloud/SKILL.md` — uses `--tags '["wake"]'` → remove

- [ ] **Step 4: Commit**

```bash
git add .claude/skills/
git commit -m "refactor(skills): update all skills for universal entries CLI

Replace routine and nutrition-specific commands with alt-db entry.
Remove --tags usage from entry add commands."
```

---

### Task 9: README, CLAUDE.md, and Rules

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `.claude/rules/database-entries.md`

- [ ] **Step 1: Update README.md**

Read `README.md`. Make these changes:

1. Add a "Why a Single Table?" section after Architecture:

```markdown
### Why a Single Table?

Most applications use purpose-built tables for each domain. alt takes a different approach: one universal `entries` table for all data.

This works because alt's "application layer" is an LLM, not compiled code. Claude Code interprets metadata flexibly based on skill instructions — it doesn't need column types to function correctly. The skill definition IS the schema.

Benefits for forkers:
- **Customize by editing skills alone** — no migrations, no schema changes
- **One table to set up** — minimal Neon configuration to get started
- **Add new data types freely** — just write a new skill with a new `type` value
```

2. Update the CLI Tools table — change `alt-db` description to: `Database CRUD for all entries (plans, goals, routines, nutrition, body metrics)`

3. Update Quick Start step 3 to mention the single entries table

- [ ] **Step 2: Update CLAUDE.md**

Read `CLAUDE.md`. Change:
```
- Routine definitions are stored in the `routines` DB table (managed via `uv run alt-db routine`)
```
to:
```
- Routine definitions are in YAML files in `data/routines/`; completion events are entries (type `routine_event`)
```

- [ ] **Step 3: Update .claude/rules/database-entries.md**

Read `.claude/rules/database-entries.md`. Remove references to specialized CLI subcommands (`routine`, `nutrition-item`, `nutrition-log`, `nutrition-target`). All operations now go through `alt-db entry`. Update the CLI Commands section and examples. Add notes about the entry types that replaced old tables:

| Old table | Entry type |
|---|---|
| routine_events | `routine_event` |
| body_measurements | `body_measurement` |
| body_measurement_goals | `body_measurement_goal` |
| nutrition_items | `nutrition_item` |
| nutrition_logs | `nutrition_log` |
| nutrition_targets | `nutrition_target` |

- [ ] **Step 4: Final verification — run all tests**

Run: `cd /Users/makoto/projects/alt && uv run pytest tests/ -v`
Expected: All tests pass

Run: `cd /Users/makoto/projects/alt/webapp && npm run build`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md .claude/rules/database-entries.md
git commit -m "docs: update README, CLAUDE.md, and rules for universal entries

Add 'Why a Single Table?' section to README. Update CLI references
and database entry documentation."
```
