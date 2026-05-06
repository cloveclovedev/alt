# Cloud-scheduler cron reconciler

Date: 2026-05-07
Status: Approved
Tracking: [#25](https://github.com/cloveclovedev/alt/issues/25)

## Goal

Drive the unified cloud-scheduler routine's cron schedule from `config`. When
the user changes a per-skill schedule param in the webapp (or via CLI), running
the reconciler updates the routine so the trigger actually fires on the right
hours. This removes the hand-edited cron line that has lived alongside the
dispatch table since the unified-trigger consolidation.

The reconciler is a Claude Code slash command. It computes the target cron
from `config`, presents the diff to the user, and — on approval — delegates the
routine update to the built-in `/schedule` skill via natural language. No
direct calls into Cron tools or `/schedule` internals.

The change also tightens the schema for cloud-execution params: each skill's
schedule param carries hour(s) only; the trigger minute is a single global
value owned by `cloud_scheduler`.

## Philosophy

- **Single source of truth.** The `config` table decides when the routine fires.
  Hand-editing the cron line in `/schedule` is no longer the supported path.
- **Trigger is coarse, dispatcher is fine.** The cron only carries the union of
  hours across all cloud skills plus the global minute. Day-of-week / day-of-month
  filtering, "is it actually time to run for this skill" gating, and any
  fallback-vs-active distinction stay in `cloud-scheduler`'s Phase 2.5 per-skill
  gate (and in each skill's own logic). The reconciler does not touch DOW/DOM.
- **No tool guesswork.** Trigger CRUD is delegated to the `/schedule` skill. The
  reconciler does not try to identify routines by name or prompt content, does
  not call `CronCreate` / `CronList` / `CronDelete` directly, and does not
  inspect `/schedule`'s implementation.
- **Confirmed before applied.** Even though the operation is idempotent, the
  reconciler shows the diff and waits for user acknowledgment before mutating
  the routine. Auto-mutation is rejected because deleting a routine drops its
  prompt; a buggy reconciler run would otherwise be silently destructive.

## Non-goals

- DOW / DOM in the cron expression. If a skill needs to run only on certain
  days, that's a per-skill gate (separate issue) — the trigger keeps firing
  daily at the union of hours and the gate exits early on non-matching days.
- Replacing Phase 2.5 hour gating in `cloud-scheduler`. The dispatcher still
  reads each skill's schedule param itself; the reconciler only ensures the
  trigger fires often enough that those gates can ever match.
- Adding cloud-schedule params for skills other than `daily-plan-cloud`.
  Migrating the rest (`x-post`, `x-draft`, `nutrition-check`, `wake-check`,
  `weekly-plan`) is per-skill and tracked separately. After this change the
  trigger will collapse to firing only at the hours covered by
  `daily_plan.cloud.fallback_hour`; that is the user-accepted intermediate
  state until other skills are migrated.
- Auto-running the reconciler from the webapp. The trigger-management
  permission surface stays inside Claude Code.
- Running the reconciler from `weekly-plan-cloud`. The cloud variant runs
  unattended and the reconciler requires user confirmation, so the auto-step
  is wired into local `/weekly-plan` only.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│  config table                                                    │
│    cloud_scheduler.cron_minute       (int, 0-59)                 │
│    daily_plan.cloud.fallback_hour    (int, 0-23)                 │
│    *.cloud.fallback_hour             (int, 0-23)                 │
│    *.cloud.run_hours                 (int[], each 0-23)          │
│    weekly_plan.reconcile_cron        (bool)                      │
└──────────┬───────────────────────────────────────────────────────┘
           │  uv run alt-db config list --with-meta --json
           ▼
┌──────────────────────────────────────────────────────────────────┐
│  alt-cron compute  (new package: src/alt_cron/)                  │
│    pure: rows + cron_minute → { hours, minute, cron, warnings }  │
└──────────┬───────────────────────────────────────────────────────┘
           │  cron string + diff
           ▼
┌──────────────────────────────────────────────────────────────────┐
│  /reconcile-cron skill                                           │
│    1. compute target                                             │
│    2. show diff vs intended baseline (see Diff section)          │
│    3. ask user to confirm                                        │
│    4. delegate to /schedule via natural-language Skill call      │
└──────────┬───────────────────────────────────────────────────────┘
           │  Skill("schedule", "<NL instruction>")
           ▼
┌──────────────────────────────────────────────────────────────────┐
│  /schedule skill (Claude Code built-in)                          │
│    handles routine list / create / update                        │
└──────────────────────────────────────────────────────────────────┘
```

The reconciler is also reachable as the last step of `/weekly-plan`, gated by
`weekly_plan.reconcile_cron` (default true). `weekly-plan-cloud` does not call
it.

## Data model changes

All changes are config-row data updates. No DB schema migration.

### New / renamed params

| Key | Type | Default | Replaces | Consumed by |
|---|---|---|---|---|
| `cloud_scheduler.cron_minute` | integer (0-59) | `23` | (new) | `cloud-scheduler`, `reconcile-cron` |
| `daily_plan.cloud.fallback_hour` | integer (0-23) | `10` | `daily_plan.cloud.fallback_time` (string `"10:23"`) | `daily-plan-cloud`, `cloud-scheduler`, `reconcile-cron` |
| `weekly_plan.reconcile_cron` | boolean | `true` | (new) | `weekly-plan` |

### Future suffix conventions (no new keys in this change)

| Suffix | Type | Meaning |
|---|---|---|
| `*.cloud.fallback_hour` | integer (0-23) | Single fallback hour. Skill runs only if the local equivalent didn't already complete. |
| `*.cloud.run_hours` | integer[] (each 0-23) | Cloud-only schedule, multiple hours per day. Not a fallback. |

Both suffix shapes contribute hours to the cron union. The reconciler scans
both and the same hour appearing in either is folded into the union.

The minute portion of any wall-clock time a skill cares about is no longer
stored in these params — the trigger fires once per hour at
`cloud_scheduler.cron_minute`.

### Migration steps (data only)

The PR that lands this change executes, in order:

1. Add `cloud_scheduler.cron_minute = 23` (with metadata `type=number`,
   `consumed_by=[cloud-scheduler, reconcile-cron]`, description).
2. Add `daily_plan.cloud.fallback_hour = 10` (metadata `type=number`,
   `consumed_by=[daily-plan-cloud, cloud-scheduler, reconcile-cron]`).
3. Delete `daily_plan.cloud.fallback_time`.
4. Add `weekly_plan.reconcile_cron = true` (metadata `type=boolean`,
   `consumed_by=[weekly-plan]`).
5. Update `.claude/config-defaults.yaml` to match (so fresh clones seed the
   new shape).

The migration is performed manually via `uv run alt-db config set`,
`set-meta`, and `delete` calls in the deployment of this change. It does not
require new tooling.

## Components

| # | Component | Purpose |
|---|---|---|
| 1 | `src/alt_cron/__init__.py`, `cli.py` | New small package. Pure function `compute_target(rows, cron_minute) -> dict` plus thin CLI `alt-cron compute` reading config-list JSON from stdin. |
| 2 | `pyproject.toml` | Register `alt-cron` entry point and add `src/alt_cron` to packaged sources. |
| 3 | `tests/test_alt_cron.py` | Pytest coverage for `compute_target`: union of `fallback_hour` + `run_hours`, dedup, sort, type / range validation, empty input error. |
| 4 | `.claude/skills/reconcile-cron/SKILL.md` | The reconciler slash command. Compute → diff → confirm → delegate to `/schedule`. |
| 5 | `.claude/skills/cloud-scheduler/SKILL.md` | Phase 2.5 gate updated to read `daily_plan.cloud.fallback_hour` (int) instead of `fallback_time` (string). |
| 6 | `.claude/skills/daily-plan/SKILL.md` (and `weekly-plan`) | At end of `/weekly-plan`, if `weekly_plan.reconcile_cron` is true, invoke `/reconcile-cron`; warn-only on failure. |
| 7 | `.claude/config-defaults.yaml` | Replace `daily_plan.cloud.fallback_time` entry; add `cloud_scheduler.cron_minute`, `daily_plan.cloud.fallback_hour`, `weekly_plan.reconcile_cron` entries. |
| 8 | `webapp/src/components/config/config-form.tsx` (or sibling) | Render an "Effective fire time" hint under any `*.cloud.fallback_hour` / `*.cloud.run_hours` field, computed from the in-form value plus `cloud_scheduler.cron_minute` from the same form load. New `cloud-scheduler` tab is auto-derived from `consumed_by`. |
| 9 | Manual data migration steps | `alt-db config set/set-meta/delete` calls listed under Migration steps, run as part of the deployment. |

## Compute logic (`alt_cron.compute_target`)

Pure function. Signature:

```python
def compute_target(rows: list[dict], cron_minute: int) -> dict
```

`rows` is the JSON list returned by `alt-db config list --with-meta --json`.
Each row has `key`, `value`, `metadata`.

Return shape:

```json
{
  "minute": 23,
  "hours": [10],
  "cron": "23 10 * * *",
  "warnings": []
}
```

Algorithm:

1. Validate `cron_minute` is integer `0..59`. Raise on violation.
2. For each row whose `key` ends in `.cloud.fallback_hour`:
   - Require `value` is integer `0..23`. Otherwise raise with the offending key.
   - Add to `hours` set.
3. For each row whose `key` ends in `.cloud.run_hours`:
   - Require `value` is a list of integers each `0..23`. Otherwise raise.
   - Add each element to `hours` set.
4. If `hours` is empty, raise: "No time-source params found; refusing to
   produce an empty cron."
5. Sort hours ascending, dedup.
6. Build `cron = f"{cron_minute} {','.join(map(str, hours))} * * *"`.
7. `warnings` is currently always empty. Reserved for future warnings (e.g.,
   when DOW params are introduced).

The function is pure: no I/O, no time, no environment. The CLI wrapper does
the I/O.

### CLI

```
$ uv run alt-db config list --with-meta --json \
    | uv run alt-cron compute --cron-minute 23
{"minute": 23, "hours": [10], "cron": "23 10 * * *", "warnings": []}
```

`--cron-minute` is required; the CLI does not read `config` itself. The
reconciler skill reads `cloud_scheduler.cron_minute` separately (for the diff
display) and passes it in.

Errors are written to stderr; exit code is non-zero on any validation failure.

## Reconciler skill flow (`.claude/skills/reconcile-cron/SKILL.md`)

### Phase 0: Environment

```bash
uv sync
```

### Phase 1: Read inputs

```bash
uv run alt-db config get cloud_scheduler.cron_minute
uv run alt-db config list --with-meta --json | uv run alt-cron compute --cron-minute <minute>
```

The output JSON gives the target cron string and the hour set.

### Phase 2: Read current routine cron

The skill invokes the `/schedule` skill via the `Skill` tool with a
natural-language instruction:

> Use the /schedule skill to list scheduled routines and report the cron
> expression of the alt cloud-scheduler routine. Reply with the cron string
> only, or "not found" if no such routine exists.

The reconciler captures the returned cron string (or "not found") for the
diff display. If `/schedule` returns ambiguous output (multiple matching
routines, parse errors), the reconciler stops with an instruction to the
user to clean up first.

### Phase 2b: Show diff

The skill prints a short, readable diff:

```
Current cloud-scheduler cron:  23 0,6,10,12,15,18,19,21 * * *
Target cloud-scheduler cron:   23 10 * * *
Hours covered:                 10
Param sources:
  daily_plan.cloud.fallback_hour = 10

Effective fire time(s) JST: 10:23
```

If "current" equals "target", the skill reports "no change" and ends here
(no Phase 3 / 4).

### Phase 3: Confirm

The skill asks the user to confirm the change before mutating. (The
no-change case has already been handled at the end of Phase 2b.)

### Phase 4: Delegate to /schedule

On approval, the skill invokes the `/schedule` skill via the `Skill` tool
with a natural-language instruction:

> Use the /schedule skill to update the alt cloud-scheduler routine to fire
> on the cron expression `23 10 * * *`. The routine's prompt should remain
> the existing one that invokes the `cloud-scheduler` skill. If a matching
> routine doesn't exist, create one with that prompt and cron.

The reconciler does not read or modify `/schedule`'s implementation. It only
relies on the natural-language interface.

### Phase 5: Report

After `/schedule` reports completion, the reconciler prints:
- the new cron and effective fire times,
- a one-line summary (e.g., "Updated cloud-scheduler routine: 23 10 * * *").

If `/schedule` reports failure, the skill surfaces the failure verbatim and
exits non-zero. No retries.

## /weekly-plan integration

At the end of the existing `/weekly-plan` skill, before its closing summary:

```
if weekly_plan.reconcile_cron is true:
    invoke `/reconcile-cron` via the Skill tool
    on success: include a one-liner in the weekly-plan summary
    on failure: surface the error in the summary, do not abort the weekly plan
```

`/weekly-plan-cloud` is unchanged — it does not call the reconciler.

## webapp

The existing generic config form already renders `metadata.type=number` as an
`<Input type="number">`, so renaming `fallback_time` (string) to
`fallback_hour` (number) does not require new component code beyond the
metadata change. Two webapp-only additions:

- **Effective fire time hint.** For any field whose key matches
  `*.cloud.fallback_hour` or `*.cloud.run_hours`, render a small caption
  underneath: `Effective fire time(s): HH:MM JST`, where MM is the value of
  `cloud_scheduler.cron_minute` from the same page load and HH(s) come from
  the form's current input. The hint updates as the user types. If
  `cloud_scheduler.cron_minute` is missing, the caption falls back to "MM
  unset — set cloud_scheduler.cron_minute first".
- **`cloud-scheduler` tab.** Adding `consumed_by: [cloud-scheduler, ...]` to
  `cloud_scheduler.cron_minute`'s metadata causes the existing
  `computeTabs` logic to surface a `cloud-scheduler` tab automatically. To
  keep Phase 1 scope intentional, `PHASE_1_SKILLS` in
  `webapp/src/app/config/page.tsx` is extended to include `cloud-scheduler`.

The webapp does not run the reconciler. Editing `fallback_hour` or
`cron_minute` saves to the DB only; the user is expected to run
`/reconcile-cron` (or wait for next `/weekly-plan`) to reflect the change in
the routine.

## Error handling

- **`cloud_scheduler.cron_minute` missing or out of range** — `alt-cron compute`
  raises with a clear message; reconciler reports the failure and stops before
  Phase 3.
- **A `*.cloud.fallback_hour` value is not an int 0..23, or `*.cloud.run_hours`
  contains an invalid element** — same: `alt-cron compute` raises pointing at
  the offending key; reconciler stops before Phase 3.
- **No time-source params found** — `alt-cron compute` raises rather than
  produce an empty cron. After this change ships there is always at least one
  (`daily_plan.cloud.fallback_hour`), so this only triggers on misconfigured
  databases. The reconciler does not delete the existing routine in this case.
- **`/schedule` reports failure (or it cannot be invoked)** — surface the
  failure verbatim; exit non-zero. `/weekly-plan` continues but shows the
  failure in its summary.
- **User cancels at Phase 3** — exit cleanly, no mutation.
- **Concurrent edits** — last-write-wins. Two reconciler runs in quick
  succession would both confirm and apply; the second one is just a no-op
  diff, which is fine.

## Testing

- **Python (`tests/test_alt_cron.py`)**:
  - empty `rows` ⇒ raises (no time-source params).
  - one `*.cloud.fallback_hour=10` ⇒ `cron == "23 10 * * *"`.
  - one `*.cloud.run_hours=[6,18]` ⇒ `cron == "23 6,18 * * *"`.
  - mix of `fallback_hour=10` and `run_hours=[6,10,18]` ⇒ hours `[6,10,18]`
    (dedup), `cron == "23 6,10,18 * * *"`.
  - hour out of range / non-int ⇒ raises with offending key in message.
  - `cron_minute` out of range ⇒ raises.
- **CLI smoke (`tests/test_alt_cron_cli.py` or extend `test_cli.py`)**:
  - `alt-cron compute --cron-minute 23` reads stdin JSON, prints valid output
    JSON for a representative input.
  - Non-zero exit on bad `--cron-minute`.
- **webapp**:
  - Manual: `npm run dev` → open `/config`, edit `fallback_hour` → caption
    updates live with `HH:23`. Switch tab to `cloud-scheduler`, change
    `cron_minute`, return to `daily-plan` and confirm caption reflects new
    minute on reload.
- **Skill (manual smoke after PR review)**:
  - Run `/reconcile-cron` once with no DB change since last run ⇒ "no change".
  - Run `/reconcile-cron` after editing `daily_plan.cloud.fallback_hour` to a
    different hour ⇒ diff shown, confirmation required, `/schedule` updates
    the routine.
  - Verify `cloud-scheduler` Phase 2.5 gate keeps working with the new int
    value (run `/cloud-scheduler` at the configured hour with mocked time).

## Open follow-ups (separate issues)

- Migrate `x-post-cloud`, `x-draft-cloud`, `nutrition-check-cloud`,
  `wake-check-cloud`, `weekly-plan-cloud` to the new schedule-param model. Each
  introduces its own `*.cloud.fallback_hour` or `*.cloud.run_hours` and updates
  the corresponding Phase 2.5 gate in `cloud-scheduler`. Until each migrates,
  the unified trigger will not fire on hours those skills currently rely on.
- Per-skill DOW / DOM gates (`*.cloud.*_days_of_week`,
  `*.cloud.*_day_of_month`) consumed by the dispatcher. Trigger and reconciler
  remain hour-only.
- Webapp UX for editing `cloud_scheduler.cron_minute` from the daily-plan tab
  inline (e.g., a read-only echo with a deep link). Currently it lives in its
  own `cloud-scheduler` tab.
