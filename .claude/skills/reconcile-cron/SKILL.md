---
name: reconcile-cron
description: "Reconcile the unified cloud-scheduler routine's cron with config — computes target hours from *.cloud.fallback_hour and *.cloud.run_hours, shows diff, and updates the routine via /schedule on confirmation"
---

# Reconcile Cron

Drives the unified cloud-scheduler routine's cron schedule from `config`.
Computes the target cron, shows the diff against the current routine, and
on user approval delegates the routine update to the `/schedule` skill.

The reconciler is **update-only**. It does not create routines from scratch
because that requires choosing the routine's prompt, which is an editorial
decision outside this skill's scope. If the routine is missing, the skill
reports it and stops.

## When invoked

### Phase 0: Environment

Install dependencies:
```bash
uv sync
```

### Phase 1: Compute target cron

Read the global cron minute and compute the target cron from time-source
config rows:

```bash
MINUTE=$(uv run alt-db config get cloud_scheduler.cron_minute)
uv run alt-db config list --with-meta --json \
  | uv run alt-cron compute --cron-minute "$MINUTE"
```

Parse the JSON output. Capture:
- `cron` (target cron string, e.g. `"23 10 * * *"`)
- `hours` (list of hours covered)
- `minute` (the cron minute)

If `alt-cron compute` exits non-zero, surface the stderr verbatim and end
the session — do not proceed to Phase 2.

### Phase 2: Read current routine cron

Invoke the `/schedule` skill via the Skill tool with this natural-language
instruction:

> Use the /schedule skill to list scheduled routines and report the cron
> expression of the alt cloud-scheduler routine. Reply with the cron
> string only, or "not found" if no such routine exists.

Capture the returned cron string (or "not found"). If the response is
ambiguous (multiple matching routines, parse errors), stop and ask the
user to clean up first.

### Phase 2b: Show diff

Print a short, readable diff:

```text
Current cloud-scheduler cron:  <current or "not found">
Target cloud-scheduler cron:   <target>
Hours covered:                 <comma-separated hours>
Param sources:
  <key1> = <value1>
  <key2> = <value2>
  ...

Effective fire time(s) JST: HH:MM, HH:MM, ...
```

Source rows for the "Param sources" section: re-scan the
`alt-db config list --with-meta --json` output for keys ending in
`.cloud.fallback_hour` or `.cloud.run_hours`.

If "current" equals "target" (string equality after trimming whitespace),
print "No change required." and end the session — skip Phase 3 and
Phase 4.

If "current" was "not found", print:
"The cloud-scheduler routine does not exist yet. Bootstrap it once via
/schedule (with a prompt that invokes the cloud-scheduler skill), then
re-run /reconcile-cron." and end the session.

### Phase 3: Confirm

Ask the user: "Apply this change? (yes/no)". If anything other than an
affirmative answer, end the session without mutation.

### Phase 4: Delegate the update to /schedule

Invoke the `/schedule` skill via the Skill tool with this instruction:

> Use the /schedule skill to update the alt cloud-scheduler routine's
> cron expression to `<TARGET>`. Keep its existing prompt unchanged.

Substitute `<TARGET>` with the target cron string from Phase 1.

If `/schedule` reports failure, surface the failure verbatim and end the
session non-zero.

### Phase 5: Report

After `/schedule` reports completion, print:

```text
Updated cloud-scheduler routine: <target cron>
Effective fire time(s) JST: HH:MM, HH:MM, ...
```

End the session.

## Notes

- The reconciler reads but never writes `cloud_scheduler.cron_minute`
  itself. To change the global minute, edit it via the webapp or
  `alt-db config set` and re-run /reconcile-cron.
- The reconciler does not handle DOW / DOM scheduling; per-skill DOW
  gates live in `cloud-scheduler`'s Phase 2.5 logic.
