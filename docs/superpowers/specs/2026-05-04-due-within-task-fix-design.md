# `--due-within` Fix: Cover task `due_date` Alongside goal `target_date`

Date: 2026-05-04

## Problem

`alt-db entry list --due-within Nd` is documented as a generic deadline filter, but in practice it only matches `goal`-style rows. Tasks with `metadata.due_date` are silently ignored.

Root cause is a single SQL fragment in `src/alt_db/entries.py:66`:

```python
conditions.append(
    f"(metadata->>'target_date')::date <= "
    f"(current_date + make_interval(days => {_next_param(params, due_within_days)}))"
)
```

The expression hard-codes `target_date`, but the personal task management spec
(`docs/superpowers/specs/2026-04-28-personal-task-management-design.md`) and
`.claude/rules/database-entries.md` define the deadline keys per type:

- `task` → `metadata.due_date`
- `goal` → `metadata.target_date`
- `body_measurement_goal` → `metadata.target_date`

The two keys are intentionally different (PM convention: `due_date` is an action-item
deadline, `target_date` is an aspirational milestone), but the filter does not know
about the split, so it never sees task deadlines.

## Goals

- Make `--due-within Nd` match rows whose effective deadline (whichever key is set)
  falls within the window.
- Keep existing goal-only callers (`daily-plan`, `daily-plan-cloud`) working
  unchanged.
- Make the deadline-key duality explicit in code so future filters / queries do not
  reintroduce the same bug.
- Add regression tests so any future change that breaks either side is caught.

## Non-Goals

- Schema normalization (collapsing `target_date` and `due_date` into one key). The
  semantic distinction is real and matches the existing spec; collapsing would
  require a data migration plus rewriting the spec, the rules doc, and every
  skill that references either key. Out of scope for a filter bug.
- Changing how skills consume the filter. `daily-plan` continues to use
  `--type goal --due-within 7d`; tasks become reachable through the same flag with
  `--type task` or no `--type`.
- Adding a separate `--task-due-within` flag. The whole point is one flag for
  "anything with a deadline coming up."

## Design

### Single deadline expression

Define a module-level constant in `src/alt_db/entries.py` that names the
effective-deadline column expression once:

```python
_DEADLINE_EXPR = "COALESCE(metadata->>'due_date', metadata->>'target_date')"
```

`COALESCE` returns the first non-null argument. `due_date` is checked first so that
if a row ever ends up with both keys (not currently expected, but possible if a
future schema lets a goal have both an aspirational and a hard date), the
task-style "hard deadline" wins, which is the safer interpretation for
"is this due within N days?".

### Filter rewrite

In `list_entries`, replace the hard-coded fragment with the expression:

```python
if due_within_days is not None:
    conditions.append(
        f"({_DEADLINE_EXPR})::date <= "
        f"(current_date + make_interval(days => {_next_param(params, due_within_days)}))"
    )
    conditions.append(f"{_DEADLINE_EXPR} IS NOT NULL")
```

The `IS NOT NULL` guard mirrors the existing implementation. It is technically
redundant given that `NULL <= x` evaluates to `NULL` (filtered out), but is kept
because (a) it matches existing code's defensive style and (b) it makes the
"need a deadline to qualify" intent explicit at the SQL level.

### Behaviour matrix

| Caller | Before | After |
|---|---|---|
| `--type goal --due-within 7d` (target_date set) | hit | hit (unchanged) |
| `--type task --due-within 7d` (due_date set) | **missed** | hit |
| `--due-within 7d` (no `--type`) | only goals | tasks + goals |
| Row with both keys set | judged by `target_date` | judged by `due_date` (preferred) |
| Row with neither key | excluded | excluded (unchanged) |

## Testing

Add three tests to `tests/test_entries.py`. They use today's date arithmetic so
they exercise the actual SQL behaviour rather than the Python wrapper.

1. `test_list_entries_due_within_task_due_date`
   - Insert a `task` with `metadata.due_date = today + 3 days`.
   - Assert it appears in `list_entries(due_within_days=7)`.
   - Asserts the fix.

2. `test_list_entries_due_within_goal_target_date`
   - Insert a `goal` with `metadata.target_date = today + 3 days`.
   - Assert it appears in `list_entries(due_within_days=7)`.
   - Regression guard: ensures the goal path still works.

3. `test_list_entries_due_within_excludes_far_future`
   - Insert a `task` with `metadata.due_date = today + 30 days`.
   - Assert it does NOT appear in `list_entries(due_within_days=7)`.
   - Regression guard: confirms the upper bound still applies.

Use `datetime.date.today() + datetime.timedelta(days=N)` for the dates so the
tests stay valid across runs. Use the `db` fixture pattern already established
in the file (`client, entry_ids = db; entry_ids.append(...)`).

## Migrations / Data Changes

None. The change is purely query-side. No existing rows are touched, no schema
columns added or removed.

## Documentation Updates

`.claude/rules/database-entries.md:215` documents:

```bash
uv run alt-db entry list --due-within 7d
```

Post-fix, this command behaves the way the documentation already implies (returns
all upcoming-deadline entries regardless of type), so no doc edit is needed.

The personal task management design doc and per-skill SKILL.md files reference
type-specific commands (`--type goal --due-within 7d`) that continue to work
unchanged.

## Rollout

- Branch: `fix/due-within-task-due-date` from `main`.
- Single commit (or two: one for the fix, one for tests) with conventional commit
  prefix `fix(alt-db):`.
- PR against `main`. No flag, no migration, no coordinated rollout needed.

## Risks

- The `::date` cast still assumes well-formed `YYYY-MM-DD` strings in metadata.
  Same risk exists today; not addressed here. If a malformed value sneaks in, the
  whole `list_entries(due_within_days=...)` query errors. Acceptable for now —
  this is internal data and the schema has been disciplined so far.
- Anyone currently relying on `--due-within` returning *only* goals (no known
  consumers do; both call sites already pass `--type goal` explicitly) will see
  task rows as well. Considered an improvement, not a regression.
