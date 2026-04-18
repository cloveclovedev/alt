# Routine CLI Enhancements — Design Spec

## Problem

The `alt-db routine` CLI currently only supports `complete`, `baseline`, `last`, and `all` commands. There is no way to:

1. **Delete** an incorrectly recorded completion/baseline
2. **Update a note** without creating a new completion record
3. **View history** of all records for a routine (only the latest is visible)

This leads to dirty data when mistakes happen (e.g., accidentally marking a routine as completed when only a note update was intended).

Additionally, the routines skill does not specify that notes must be written in English, leading to inconsistency.

## Design

### New CLI Commands

#### `routine history <name>`

List all records for a given routine, newest first, with UUIDs visible for subsequent delete/update operations.

**CLI usage:**
```
uv run alt-db routine history "<name>"
```

**Output format (text):**
```
ID                                    Date         Kind       Note
a1b2c3d4-e5f6-7890-abcd-ef1234567890  2026-04-11  completed  —
f9e8d7c6-b5a4-3210-fedc-ba0987654321  2026-03-01  completed  Deep cleaned
```

**Output format (JSON):** Array of event objects (same shape as `routine last`).

**DB function:** `get_history(db: NeonHTTP, name: str) -> list[dict]`
- Query: `SELECT id, routine_name, category, completed_at, kind, note FROM routine_events WHERE routine_name = $1 ORDER BY completed_at DESC`
- Returns all records, not just the latest

#### `routine delete <id>`

Delete a single routine event by UUID.

**CLI usage:**
```
uv run alt-db routine delete <uuid>
```

**Output:** `Deleted routine event <uuid>` on success, exit 1 with error message if not found.

**DB function:** `delete_event(db: NeonHTTP, event_id: str) -> bool`
- Query: `DELETE FROM routine_events WHERE id = $1`
- Returns True if a row was deleted, False otherwise

#### `routine update-note <id> --note "..."`

Update only the note field of an existing routine event.

**CLI usage:**
```
uv run alt-db routine update-note <uuid> --note "New note text"
```

**Output:** `Updated note for <uuid>` on success, exit 1 with error message if not found.

**DB function:** `update_note(db: NeonHTTP, event_id: str, note: str | None) -> bool`
- Query: `UPDATE routine_events SET note = $1 WHERE id = $2`
- Empty string `--note ""` sets note to NULL
- Returns True if a row was updated, False otherwise

### Skill Updates (`routines/skill.md`)

**1. Note language rule** — Add to section 7 (Interactive actions):

> All notes MUST be written in English.

**2. Error correction flow** — Add new section 8:

> **Correcting mistakes:**
> When a routine is incorrectly marked as completed, or a note needs correction:
> 1. Run `uv run alt-db routine history "<name>"` to list all records with IDs
> 2. Run `uv run alt-db routine delete <id>` to remove incorrect records
> 3. Run `uv run alt-db routine update-note <id> --note "..."` to fix notes on existing records

### Files Changed

| File | Change |
|---|---|
| `src/alt_db/routines.py` | Add `get_history`, `delete_event`, `update_note` |
| `src/alt_db/cli.py` | Add `routine history`, `routine delete`, `routine update-note` subcommands |
| `tests/test_routines.py` | Tests for new functions |
| `.claude/skills/routines/skill.md` | Note language rule + error correction flow |

### No Changes

- **DB schema**: No migration needed. Existing `routine_events` table and columns are sufficient.
- **Existing commands**: `complete`, `baseline`, `last`, `all` remain unchanged.
