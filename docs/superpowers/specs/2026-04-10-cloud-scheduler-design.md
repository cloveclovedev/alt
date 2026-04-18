# Unified Cloud Scheduler Design

## Problem

Cloud scheduled tasks have a per-plan limit of 3 triggers. All 3 slots are currently used (Daily Planner, Weekly Planner, X Draft Generator), leaving no room for new skills like `nutrition-check-cloud` (5 daily runs).

## Solution

Create a single unified dispatcher skill (`cloud-scheduler`) that runs at all required times and routes to the appropriate skills based on current JST hour and day of week.

## Dispatch Table

| JST Hour | Day    | Skills (execution order)                                                      |
|----------|--------|-------------------------------------------------------------------------------|
| 0        | Daily  | nutrition-check-cloud                                                         |
| 6        | Daily  | nutrition-check-cloud                                                         |
| 10       | Sun    | weekly-plan-cloud → daily-plan-cloud → x-draft-cloud → nutrition-check-cloud  |
| 10       | Mon-Sat| daily-plan-cloud → x-draft-cloud → nutrition-check-cloud                      |
| 15       | Daily  | nutrition-check-cloud                                                         |
| 18       | Daily  | x-draft-cloud                                                                 |
| 21       | Daily  | nutrition-check-cloud                                                         |

## Cron Expression

`0 1,6,9,12,15,21 * * *` (UTC)

JST-to-UTC mapping:

| JST   | UTC   |
|-------|-------|
| 00:00 | 15:00 |
| 06:00 | 21:00 |
| 10:00 | 01:00 |
| 15:00 | 06:00 |
| 18:00 | 09:00 |
| 21:00 | 12:00 |

## Architecture

### Dispatcher Skill (`cloud-scheduler/SKILL.md`)

A pure routing layer with no business logic. Responsibilities:

1. Get current JST time (`TZ=Asia/Tokyo date`)
2. Match current hour + day_of_week against the dispatch table
3. Invoke each matched skill sequentially via the `Skill` tool
4. If a skill fails, log the error and continue with the remaining skills
5. If no slot matches the current hour (e.g., cron drift), log and exit silently

### Execution Model

- **Skill tool dispatch**: The dispatcher invokes sub-skills using the `Skill` tool. All skills run in the same agent context sequentially.
- Each sub-skill's SKILL.md content is loaded into the conversation context when invoked.
- Context accumulation is acceptable (~20KB for 4 skills at the 10:00 slot, negligible against 1M context).

### Error Handling

- Each skill invocation is independent. A failure in one skill does not block subsequent skills.
- The dispatcher logs which skills succeeded and which failed before exiting.

## Trigger Management

### Delete (3 existing triggers)

| Trigger ID                        | Name            | Cron              |
|-----------------------------------|-----------------|--------------------|
| `trig_016sMVEwwxntcxhrV9EqxCPt`  | Daily Planner   | `0 1 * * *`       |
| `trig_01GVxDaPNTpcGK4L3ev6m9z6`  | Weekly Planner  | `30 0 * * 0`      |
| `trig_01TAUVje3QbvbrVUBhLqbWsA`  | X Draft Generator | `30 2,9 * * *`  |

### Create (1 new trigger)

| Name             | Skill              | Cron                          |
|------------------|--------------------|-------------------------------|
| Cloud Scheduler  | /cloud-scheduler   | `0 1,6,9,12,15,21 * * *`     |

## Existing Skills

**No changes required.** All existing cloud skills remain functional both as dispatcher targets and as standalone skills (invocable locally via `/daily-plan-cloud`, etc.).

## Out of Scope

- `wake-check-cloud`: Requires sub-hour escalation intervals (every 10 min). Will use a separate dedicated trigger when implemented.
