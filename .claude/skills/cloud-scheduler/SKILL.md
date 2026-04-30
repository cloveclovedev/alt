---
name: cloud-scheduler
description: "[Cloud scheduled task] Unified dispatcher - routes to appropriate skills based on JST time"
---

# Cloud Scheduler

Unified dispatcher for all cloud scheduled tasks. Determines which skills to run based on the current JST hour and day of week, then invokes them sequentially.

## When invoked

### Phase 0: Environment

Install dependencies:
```bash
uv sync
```

### Phase 1: Determine Current Time

Get the current JST hour and day of week:
```bash
TZ=Asia/Tokyo date '+%H %u'
```
This outputs `<hour> <day_of_week>` where hour is 00-23 and day_of_week is 1 (Monday) through 7 (Sunday).

Parse the output into `<hour>` (integer) and `<day_of_week>` (integer).

### Phase 2: Dispatch

Match the current hour and day against the dispatch table below. Invoke each skill **sequentially** using the `Skill` tool.

The trigger fires 8 times per day at JST 0/6/10/12/15/18/19/21 (HH:23). x-post-cloud runs first at every slot; other skills run at their designated slots.

| Hour | Day     | Skills (invoke in this order)                                                                     |
|------|---------|---------------------------------------------------------------------------------------------------|
| 0    | Daily   | x-post-cloud → nutrition-check-cloud                                                              |
| 6    | Daily   | x-post-cloud → nutrition-check-cloud                                                              |
| 10   | Sun (7) | x-post-cloud → weekly-plan-cloud → daily-plan-cloud → x-draft-cloud → nutrition-check-cloud       |
| 10   | 1-6     | x-post-cloud → daily-plan-cloud → x-draft-cloud → nutrition-check-cloud                           |
| 12   | Daily   | x-post-cloud                                                                                      |
| 15   | Daily   | x-post-cloud → nutrition-check-cloud                                                              |
| 18   | Daily   | x-post-cloud → x-draft-cloud                                                                      |
| 19   | Daily   | x-post-cloud                                                                                      |
| 21   | Daily   | x-post-cloud → nutrition-check-cloud                                                              |

For each skill in the matched row:
1. Invoke it using the `Skill` tool with the skill name (e.g., `x-post-cloud`)
2. Follow the loaded skill's instructions completely through all phases
3. If the skill encounters an error, note the failure and proceed to the next skill in the row

### Phase 2.5: Per-skill gates

Some skills now read their own enable/time configuration from the `config`
table. Apply these gates *before* invoking the relevant skill, regardless of
the dispatch table:

**daily-plan-cloud** — read both keys:
```bash
uv run alt-db config get daily_plan.cloud.enabled
uv run alt-db config get daily_plan.cloud.fallback_time
```

If `daily_plan.cloud.enabled` is `false`, skip daily-plan-cloud.
If the hour part of `daily_plan.cloud.fallback_time` (e.g. `10` from `"10:23"`)
does not match `<hour>` from Phase 1, skip daily-plan-cloud.

(All other skills retain their existing dispatch-table behaviour for now.
Per-skill gates for them will be added in subsequent issues.)

### Phase 3: Summary

After all dispatched skills have completed, output a brief summary:
- Which skills were invoked
- Which succeeded and which failed (if any)

End the session.
