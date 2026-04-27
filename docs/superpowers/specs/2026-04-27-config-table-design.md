# Config Table — Design Document

**Date**: 2026-04-27
**Status**: Approved (brainstorming complete)
**Issue**: [#11](https://github.com/cloveclovedev/alt/issues/11)

## Problem

`alt.toml` holds personal app configuration (Discord channel IDs, GitHub repos, calendar context, wake/sleep settings, nutrition coefficients, etc.). The file is gitignored to keep personal values out of the public OSS repo, but this creates friction:

- Every contributor / new install must hand-craft `alt.toml` from `alt.toml.example`.
- The webapp cannot share the same source of truth.
- Some personal settings leak into `.claude/skills/*/SKILL.md` files instead of being centralized.
- `routine_definition` records sit in `entries` despite being conceptually closer to "definitions" than "accumulating events".

Goal: move all behavior-affecting configuration to the database. Keep `.env*` files for credentials only.

## Decision

Introduce a single `config` table with a key-value JSONB schema. Reuse `entries` strictly for accumulating data (events, logs, notes). Eliminate `alt.toml` entirely.

### Conceptual model

| Table | Purpose | Examples |
|---|---|---|
| `entries` | Append-only / accumulating data with chronological value | `routine_event`, `nutrition_log`, `body_measurement`, `memo`, `knowledge`, `tech_interest`, `business`, `goal` |
| `config` | Single current-value records (settings + user-defined definitions) | `discord.daily_channel_id`, `wake.escalation.interval_minutes`, `routines`, `nutrition_targets` |

`goal` stays in `entries` because its lifecycle (`active` → `achieved` / `abandoned`) gives historical value to past instances; a PRIMARY KEY constraint in `config` would prevent multiple states for the same key.

## Schema

```hcl
# db/schema/config.hcl
table "config" {
  schema = schema.public

  column "key" {
    type = text
    null = false
  }
  column "value" {
    type = jsonb
    null = false
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
    columns = [column.key]
  }
}
```

SQL equivalent:

```sql
CREATE TABLE config (
  key        TEXT PRIMARY KEY,
  value      JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Notes:**
- `key` is `PRIMARY KEY (TEXT)` — uniqueness guaranteed by the DB; prefix queries (`WHERE key LIKE 'discord.%'`) use the PK index.
- `value` is `JSONB NOT NULL` — accepts any JSON type (string, number, boolean, array, object).
- `updated_at` is set explicitly by application code on UPDATE (mirrors the existing `entries` pattern; no trigger).

## Key naming convention

**Rule:** keys follow `<function>.<integration>.<param>` (or `<function>.<param>` for function-internal settings). Dot-notation flat keys for atomic settings; single key with a JSONB object for user-defined collections.

The top-level segment names a feature area:
- `core` — cross-cutting setup shared by multiple features
- `plan` — daily/weekly planning workflow
- `draft` — content drafting (currently `draft.x.*` for X posts)
- `nutrition` — nutrition tracking
- `wake` — wake-up / sleep schedule
- `body` — body composition tracking

The second segment, when present, names the integration the value belongs to (`discord`, `github`, `google_calendar`, `home_assistant`, `x`).

### Atomic settings (fixed schema) → granular dot keys

```
plan.discord.channel_id                  -> "1410506698967220224"
plan.github.repos                        -> ["cloveclovedev/alt", ...]
plan.google_calendar.context             -> "..."

core.timezone                            -> "Asia/Tokyo"
core.home_assistant.url                  -> "http://homeassistant.local:8123"
core.home_assistant.tts_entity           -> "media_player.living_room"

draft.x.post_times                       -> ["12:00", "19:00"]
draft.x.discord.channel_id               -> "..."         # where drafts are posted
draft.x.discord.input_channel_ids        -> ["..."]       # source memo channels (array)

body.height_m                            -> 1.73

wake.wakeup_time                         -> "06:30"
wake.google_calendar.adaptive            -> true
wake.prep_minutes                        -> 60
wake.escalation.interval_minutes         -> 10
wake.escalation.max_attempts             -> 3
wake.night.bedtime                       -> "23:00"
wake.night.google_calendar.lookahead     -> true

nutrition.discord.channel_id             -> "..."
nutrition.protein_coefficient            -> 2.0
nutrition.activity_factor                -> 1.5
nutrition.lean_bulk_surplus_kcal         -> 200
```

Independent atomic settings — granular updates make sense; deep keys map directly to skill / CLI lookups.

### User-defined collections → single key with JSONB object

```
draft.x.product_links     -> { "peppercheck": "https://...", "alt": "https://..." }
routines                  -> { "Clean the toilet": {...}, "朝の歯磨き": {...} }
nutrition_targets         -> { "daily": {...} }
body_measurement_goals    -> { "weight": {...} }
```

User-chosen names of arbitrary count → store the whole collection as one JSONB object. Adding / removing items = read-modify-write the whole value.

### `routines` value structure

The `routines` collection key uses the routine's display name as its inner key. This preserves the current matching mechanism in the `routines` skill (where `routine_event.title` matches `routine_definition.title`).

```json
{
  "Clean the toilet": {
    "category": "household",
    "interval_days": 14,
    "available_days": ["sat", "sun"],
    "content": "Requires brush replacement quarterly",
    "status": "active"
  },
  "朝の歯磨き": {
    "category": "hygiene",
    "interval_days": 1,
    "status": "active"
  }
}
```

**Title-as-key rationale:**
- Existing `entries(type='routine_event')` records keep working unchanged (matched by `title`).
- The skill's lookup logic only changes its source: `entry list --type routine_definition` → `config get routines`.
- Snapshot semantics remain valid: a past `routine_event.title` is the routine's name at the time of completion. If the routine is later renamed in `config.routines`, past events keep the old name as historical record.

### `routine_event` (entries) — unchanged

```
entries(
  type='routine_event',
  title='Clean the toilet',           -- matches config.routines key
  status='completed',
  metadata={
    'category': 'household',
    'completed_at': '2026-04-27T07:00:00+09:00'
  }
)
```

No schema or data change to `entries.routine_event` is required.

## CLI

`alt-db config` subcommands, following `alt-db entry` patterns:

```bash
# Get a value (deserialized JSON to stdout)
uv run alt-db config get discord.daily_channel_id

# Set a value (input is always a JSON literal)
uv run alt-db config set discord.daily_channel_id '"123456"'
uv run alt-db config set wake.calendar_adaptive 'true'
uv run alt-db config set wake.prep_minutes '60'
uv run alt-db config set github.repos '["repo1","repo2"]'

# Set from a file (for large JSON like routines)
uv run alt-db config set routines --from-file routines.json

# List (optional prefix filter)
uv run alt-db config list
uv run alt-db config list --prefix discord.

# Delete
uv run alt-db config delete x.product_links.old_product
```

**Design choices:**
- Values are always JSON literals on the CLI → explicit type, no coercion ambiguity (`"123"` vs `123`).
- `set` replaces the entire value at the key. Partial / patch updates are out of scope (YAGNI).
- No `import` command. `alt.toml` is removed entirely; values are set manually or via the migration script for `routines`.

## Python helper module

`src/alt_db/config.py`, mirroring `entries.py`:

```python
import json
from .connection import NeonHTTP

def get(db: NeonHTTP, key: str, default=None):
    """Get a config value by key. Returns the deserialized JSON value, or default if not found."""
    result = db.execute("SELECT value FROM config WHERE key = $1", [key])
    if not result.rows:
        return default
    value = result.rows[0][0]
    return json.loads(value) if isinstance(value, str) else value

def set(db: NeonHTTP, key: str, value) -> None:
    """Upsert a config value. Value is auto-JSON-encoded."""
    db.execute(
        """INSERT INTO config (key, value) VALUES ($1, $2::jsonb)
           ON CONFLICT (key) DO UPDATE
             SET value = EXCLUDED.value, updated_at = now()""",
        [key, json.dumps(value)]
    )

def list_configs(db: NeonHTTP, prefix: str | None = None) -> list[dict]:
    """List config entries, optionally filtered by key prefix."""
    if prefix is not None:
        result = db.execute(
            "SELECT key, value, created_at, updated_at FROM config WHERE key LIKE $1 ORDER BY key",
            [f"{prefix}%"],
        )
    else:
        result = db.execute(
            "SELECT key, value, created_at, updated_at FROM config ORDER BY key"
        )
    return [_row_to_dict(row) for row in result.rows]

def delete(db: NeonHTTP, key: str) -> bool:
    """Delete a config entry. Returns True if deleted."""
    result = db.execute("DELETE FROM config WHERE key = $1", [key])
    return result.row_count > 0
```

Python CLIs (`alt_body/cli.py`, `alt_home_assistant/cli.py`) replace `_find_config()` with `config.get(db, "<dot.key>")`.

## Migration plan

### 1. Schema migration

Add `db/schema/config.hcl` and run:

```bash
atlas migrate diff add_config_table --revisions-schema public
```

Generates a new SQL file under `db/migrations/`. Apply via the existing `atlas migrate apply` flow.

### 2. Manual config population

After deploying the schema, populate the existing `alt.toml` values via `alt-db config set` invocations. No `import` command is provided — values are personal and the user knows them.

### 3. Routine definition migration script

One-shot script at `.superpowers/scripts/migrate_routines.py` (gitignored — runs once, then deleted):

```python
"""One-shot: migrate entries(type='routine_definition') -> config.routines"""
from alt_db.connection import NeonHTTP
from alt_db import entries, config

db = NeonHTTP.from_env()
defs = entries.list_entries(db, type="routine_definition")

routines = {}
for entry in defs:
    value = dict(entry.get("metadata") or {})  # preserve all metadata
    if entry.get("content") is not None:
        value["content"] = entry["content"]
    if entry.get("status") is not None:
        value["status"] = entry["status"]
    routines[entry["title"]] = value

config.set(db, "routines", routines)

# Verification — bail before deleting if anything drifted
written = config.get(db, "routines")
for entry in defs:
    assert entry["title"] in written, f"missing: {entry['title']}"
    written_value = written[entry["title"]]
    for key, expected in (entry.get("metadata") or {}).items():
        assert written_value.get(key) == expected, f"metadata drift: {entry['title']}.{key}"
    if entry.get("content") is not None:
        assert written_value.get("content") == entry["content"], f"content drift: {entry['title']}"
    if entry.get("status") is not None:
        assert written_value.get("status") == entry["status"], f"status drift: {entry['title']}"
print(f"Verified {len(defs)} routines")

# Bulk delete after verification passes
for entry in defs:
    entries.delete_entry(db, entry["id"])
print(f"Deleted {len(defs)} routine_definition entries")
```

### 4. Skill updates

For each `.claude/skills/*/SKILL.md` referencing `alt.toml`, replace with `uv run alt-db config get <dot.key>` invocations:

- `daily-plan`, `daily-plan-cloud`, `weekly-plan`, `weekly-plan-cloud` — calendar context, GitHub repos, daily channel
- `wake-check-cloud` — wake settings
- `nutrition-check-cloud` — nutrition channel + coefficients
- `x-draft-cloud`, `x-post-cloud` — draft channel, product_links
- `routines` — `entry list --type routine_definition` → `config get routines`

### 5. Python CLI updates

- `src/alt_body/cli.py`: replace `_find_config()` with `config.get(db, "body.height_m")`
- `src/alt_home_assistant/cli.py`: replace `_find_config()` with `config.get(db, "home_assistant.tts_entity")` etc.

### 6. Cleanup

- Delete `alt.toml` and `alt.toml.example`.
- Remove `alt.toml` from `.gitignore`.
- Update `README.md` (drop `cp alt.toml.example alt.toml`; document `alt-db config set` workflow).
- Update `CLAUDE.md` (remove `alt.toml` references).

## Out of scope

- **Webapp settings page** (issue #11 checklist item) — separate follow-up issue.
- **Atomic patch / merge updates** for collection values (`config patch routines.X`) — add only when needed.
- **Validation / schema enforcement** on values — not required for personal use; revisit if drift becomes a problem.
- **Audit / history of config changes** — `entries` is the place if needed; current values in `config` are deliberately snapshot-only.

## Open questions / future work

- If routine renames become common, revisit title-as-key in favor of slugified keys + a separate `title` field.
- If many user-defined collections emerge, consider an `alt-db config patch <collection>.<item> '<json>'` helper that does atomic read-modify-write.
