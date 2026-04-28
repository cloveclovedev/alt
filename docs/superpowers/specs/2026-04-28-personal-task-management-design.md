# Personal Task Management Design

- Status: Draft
- Date: 2026-04-28
- Issue: [#13](https://github.com/cloveclovedev/alt/issues/13)

## Background

`alt` was published as a public OSS repo (`cloveclovedev/alt`). Until now, the project has used GitHub issues for all task management — both project work and life/daily tasks (per `.claude/rules/issue-management.md`). With the public repo, life tasks can no longer live in GitHub issues, so they need a new home.

Three storage options were considered: an external system (e.g., Google Tasks), a private GitHub repo, or the existing `entries` table in the alt DB. The DB option was selected because:

- The user already captures notes on mobile via Discord. Mobile task entry through a separate native app (Google Tasks) would split the capture flow without clear gain.
- Daily/weekly plan skills already query `entries` for goals, memos, and routine completions. Adding `type='task'` keeps planning integrated.
- The `entries` table absorbed routines (#14/#15) on the same "consolidate generic types via JSON metadata" philosophy. Tasks fit the same shape.

## Decision Summary

- New entry type `task` in the existing `entries` table. No schema migration required.
- Mobile capture continues via Discord. Daily/weekly plan skills mediate "Discord memo → DB task" promotion interactively.
- daily-plan and daily-plan-cloud surface notable tasks in the channel summary (📌 icon) and the full active list in the thread detail.
- weekly-plan and weekly-plan-cloud surface the active list and a backlog review section in the (single) post body.
- `.claude/rules/database-entries.md` is restructured to document each entry type's schema in a single per-type reference. Existing types are backfilled from current usage as part of this work.

## Data Model: `type='task'`

| Field | Value |
|---|---|
| `type` | `task` |
| `title` | Task title (English, per entries language rule) |
| `content` | Free-form notes, optional |
| `status` | `active` (default), `backlog`, `done`, `cancelled` |
| `metadata` | `{ priority?: "P0"\|"P1"\|"P2"\|"P3", due_date?: "YYYY-MM-DD" }` |
| `parent_id` | Optional. References a parent task entry id for sub-tasks. |

### Status semantics

- `active` — selected to work on. Surfaces in daily-plan active list and (when notable) in summary.
- `backlog` — captured but not committed to do soon. Reviewed in weekly-plan; may be promoted to `active` or marked `cancelled`.
- `done` — completed.
- `cancelled` — deliberately not doing. Distinct from `done`.

### Sub-task semantics

- Parent and child statuses are independent. Marking the parent `done` does not change child statuses; the user marks remaining children explicitly.
- Display: daily-plan and weekly-plan render sub-tasks indented one level under their parent.

### Promotion criteria for "notable" elevation

A task qualifies for the daily-plan channel summary if `status='active'` AND (`metadata.due_date <= today` OR `metadata.priority` is `P0` or `P1`). The summary is capped at 2-3 task lines; remaining active tasks live in the thread detail.

## Discord → DB Promotion Flow

The promotion flow runs only in the interactive variants (`daily-plan`, `weekly-plan`). Cloud variants do NOT register tasks autonomously — they only surface candidate memos in the post for the next interactive run to handle.

Interactive flow:

1. Skill reads recent Discord memos via `alt-discord read` (already done for context in current skills).
2. Skill loads existing tasks for duplicate check:
   ```bash
   uv run alt-db --json entry list --type task --status active
   uv run alt-db --json entry list --type task --status backlog
   ```
3. For each memo line that the skill judges to be a candidate task (semantic, not pattern-based):
   - If a similar existing task is found by title/content, surface it: "Existing task `X` looks similar — register new, append note to existing, or skip?"
   - If no similar task exists, propose registration with a draft title/status/priority/due_date.
4. On user approval, register:
   ```bash
   uv run alt-db entry add --type task \
     --title "<title>" \
     --status <active|backlog> \
     --metadata '{"priority": "...", "due_date": "..."}'
   ```

Cloud variants (`daily-plan-cloud`, `weekly-plan-cloud`):

- Do NOT call `entry add` for tasks. Reason: autonomous registration risks spurious entries from misjudged memos, and the user has no chance to correct.
- MAY include a "Possible task candidates" mention in the post body for clearly actionable memos, so the user can register them on the next interactive run.
- Same applies to backlog promotion in weekly-plan-cloud — backlog items are listed but never moved to `active` autonomously.

## daily-plan / daily-plan-cloud Updates

Phase 1 (data collection) gains:

```bash
uv run alt-db --json entry list --type task --status active
```

Phase 2/3 (presentation, interactive only) include the promotion flow above before generating output.

Phase 4 (post to Discord) — both variants:

Channel summary template gains a 📌 line per notable task (max 2-3 lines):

```
📋 <YYYY-MM-DD> (<Day>) Daily Plan

🔧 repo#123: issue title
📅 10:00 Calendar event
✅ Routine item
📌 Task title — due today (P1)
```

Thread detail gains a section between Recommended Issues and Routines Due:

```
## Tasks (Active)
- [P1] Title (due 2026-04-28)
  - Sub-task: Title
- [P2] Title
- ...
```

Active tasks are sorted by priority (P0 → P3, then unprioritized) then by `due_date` (earliest first, no-due last). Sub-tasks render indented under their parent regardless of their own priority.

## weekly-plan / weekly-plan-cloud Updates

Both variants currently post a single Discord message (no thread split). This design keeps that structure and adds two sections to the post body:

```
## Tasks (Active)
- [P1] Title (due 2026-04-28)
- ...

## Backlog Review
- Title (added 2026-04-15)
- ...
```

Phase 1 (data collection) gains:

```bash
uv run alt-db --json entry list --type task --status active
uv run alt-db --json entry list --type task --status backlog
```

Phase 3 (Interactive Goal Setting, interactive only) gains:

- Discord recent-memo promotion (same as daily-plan, but spanning the past week).
- Backlog review: for each backlog task, prompt the user to promote to `active` for the upcoming week, keep in `backlog`, or mark `cancelled`.

Channel summary / thread detail restructuring of weekly-plan is out of scope — tracked separately.

## Documentation: Per-Type Entries Reference

`.claude/rules/database-entries.md` is restructured into:

1. Overview (existing type table)
2. Per-Type Reference (new)
3. CLI Commands (existing, unchanged)
4. When to Save / When NOT to Save (existing)
5. Language (existing)

Each per-type section follows this format (no bold field labels — the bullet/colon structure already separates label from value):

```
### task
- Status: active(default) / backlog / done / cancelled
- Metadata: { priority: "P0|P1|P2|P3" (optional), due_date: "YYYY-MM-DD" (optional) }
- Content: free-form notes (optional)
- Parent: optional — references parent task entry for sub-tasks
- Lifecycle: Discord memo → daily-plan/weekly-plan promotion → active or backlog → done or cancelled
- Consumers: daily-plan (active + notable to summary), weekly-plan (active + backlog review), webapp /entries
```

Backfill targets (current usage extracted from skills/code):

- `task` (new)
- `knowledge` / `goal` / `memo` / `tech_interest` / `business`
- `routine_event` (per `.claude/skills/routines/SKILL.md`)
- `body_measurement` / `body_measurement_goal` (per `webapp/src/components/body/`)
- `nutrition_item` / `nutrition_log` / `nutrition_target` (per nutrition-check-cloud skill)
- `daily_plan` / `weekly_plan` (per daily-plan/weekly-plan post-save step)

CLI examples are kept only in the existing CLI Commands section, not duplicated per type.

## Migration

No migration of existing GitHub issues is required. The current `cloveclovedev/alt` open issues are project work (alt features), not life tasks; they remain in GitHub. Life-task entries are added going forward, mostly via the Discord promotion flow.

If specific life tasks are already tracked elsewhere (memory, calendar) and warrant an initial backlog seed, the user adds them manually via `alt-db entry add --type task` after rollout.

## Out of Scope

- Dedicated CLI subcommands `alt-db task ...` (revisit if generic `entry` ergonomics become a friction point).
- Dedicated webapp `/tasks` page (`/entries?type=task&status=active` covers it for now).
- Google Calendar sync of task `due_date` (separate issue if needed).
- Restructuring weekly-plan output into channel summary / thread detail (separate issue).
- Migrating existing GitHub issues into task entries.

## Open Questions

None at design time.
