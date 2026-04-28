---
name: daily-plan
description: Use when starting the day and need to plan — gathers calendar, GitHub issues, routines, and Discord context
---

# Daily Planning

## When invoked

### Phase 1: Gather Information

First, determine today's date in JST (do NOT rely on the system-provided `currentDate` which may be UTC):
```bash
TZ=Asia/Tokyo date +%Y-%m-%d
```
Use this date as `<today>` for all subsequent queries. Also compute `<sunday>` (end of the current week) from this date.

Run these commands in parallel to collect today's context:

1. **Google Calendar (today + rest of week):**
   Use Google Calendar MCP connector tools:
   - `list_calendars` to get all calendars
   - `list_events` for each calendar (skip "Holidays in Japan" and "Weather"), timeMin=`<today>T00:00:00+09:00`, timeMax=`<sunday>T23:59:59+09:00`
   - Apply calendar context rules from `plan.google_calendar.context` (read via `uv run alt-db config get plan.google_calendar.context`)

   **Calendar notes:**
   - The "Event" calendar is a memo/reminder calendar for optional activities (e.g., basketball open gym, movie discount days). Do not include these in the main schedule — list them separately as optional items in "Rest of Week Overview" or mention as a brief reminder when relevant to today.

2. **GitHub Issues:**
   Read GitHub repos: `uv run alt-db config get plan.github.repos`. For each repo:
   ```bash
   gh issue list --repo <repo> --state open --json number,title,labels,milestone,updatedAt
   ```

3. **Overdue Routines:**
   Run the routines skill logic (`uv run alt-db config get routines` for definitions + `uv run alt-db --json entry list --type routine_event` for completions, deduplicate by title keeping latest per routine) to identify overdue and due-soon routines.

4. **Discord Recent Notes (optional):**
   If a Discord bot is configured, check recent messages in the daily report channel for context from previous days.

5. **Knowledge Store Context:**
   Gather relevant entries from the knowledge store:
   - Active goals: `uv run alt-db entry list --type goal --status active`
   - Recent memos (last 7 days): `uv run alt-db entry list --since 7d`
   - Goals with approaching deadlines: `uv run alt-db entry list --type goal --due-within 7d`

6. **Tasks (Active and Backlog):**
   ```bash
   uv run alt-db --json entry list --type task --status active
   uv run alt-db --json entry list --type task --status backlog
   ```
   These are personal tasks tracked in the entries table (separate from GitHub issues). Sort active by priority (P0→P3, then unprioritized) then by `metadata.due_date` (earliest first, no-due last). Backlog is used for duplicate detection during the Phase 3 promotion flow.

### Phase 2: Present Summary

Present all gathered information organized as:

```
## Today's Schedule (YYYY-MM-DD Day)
- HH:MM-HH:MM [Calendar] Event name
- ...

## Recommended Issues (GitHub)
Select up to 5 recommended issues based on priority (P0/P1 first), milestone urgency, and recent activity. Do not list all open issues — the weekly plan covers that.
- repo#123: Issue title [P1]
- ...

## Tasks (Active)
- [P1] Title (due 2026-04-28)
  - Sub-task: Title
- [P2] Title
- ...

## Routines Due
- Overdue: ...
- Due soon: ...

## Goals & Reminders
- Active goals with status
- Goals with approaching deadlines (flagged)
- Recent memos for context

## Recent Notes
- (from Discord daily channel)

## Rest of Week Overview
- ...
```

### Phase 3: Interactive Planning

Discuss with the user:
- What are today's priorities?
- Any tasks to defer or add?
- Time blocking suggestions based on calendar gaps
- Which routines to handle today?
- Promote Discord memos to tasks: review the recent memos collected in Phase 1 step 4. For each memo line that looks like an actionable task:
  1. Check existing tasks for similar titles via `uv run alt-db --json entry list --type task --status active` and `--status backlog` (already collected in step 6) — judge similarity semantically (not pattern-based).
  2. If similar exists, ask: "Existing task `<title>` looks similar — register new, append note to existing, or skip?"
  3. If no similar task exists, propose registration with draft title/status/priority/due_date and confirm with the user.
  4. On approval, register: `uv run alt-db entry add --type task --title "<title>" --status <active|backlog> --metadata '{"priority":"...","due_date":"..."}'`

### Phase 4: Post to Discord

After the planning discussion is complete, **always** post the finalized plan to Discord. Do not ask — just post it.

Read daily channel: `uv run alt-db config get plan.discord.channel_id`.

**Step 1: Generate the summary**

Based on the Phase 3 discussion outcome, generate a concise channel summary:

```
📋 <YYYY-MM-DD> (<Day>) Daily Plan

🔧 repo#123: issue title
🔧 repo#456: issue title
📅 Notable calendar event HH:MM
✅ Routine item
📌 Task title — due today (P1)
```

- **🔧** Issues decided to work on today
- **📅** Calendar events requiring action or attention
- **✅** Routines to handle today
- **📌** Notable tasks: status=active AND (due_date<=today OR priority in [P0,P1]). Cap at 2-3 lines; remaining active tasks live in the thread detail.
- One item per line, icon repeated per item
- Omit categories with no items
- Blank line between title and items

**Step 2: Generate the full plan**

The full plan reflects the discussion outcome (revised schedule, priorities, notes). Use these sections:

- **Today's Schedule** — full calendar event list
- **Recommended Issues** — curated issue candidates with priority labels
- **Tasks (Active)** — sorted by priority then due_date; sub-tasks indented under their parent
- **Routines Due** — overdue and upcoming
- **Goals & Reminders** — active goals, approaching deadlines, memos
- **Rest of Week Overview** — upcoming events and deadlines

**Step 3: Post summary to channel**

```bash
uv run alt-discord post "<daily_channel_id>" "<summary_text>"
```

Parse the JSON output to extract `message_ids[0]` as `<summary_message_id>`.

**Step 4: Post full plan to thread**

```bash
uv run alt-discord post-thread "<daily_channel_id>" "📋 <YYYY-MM-DD> (<Day>) Daily Plan" "<full_plan_text>" --message-id <summary_message_id>
```

Parse the JSON output to extract `thread_id`.

**Step 5: Save to DB**

Combine summary and full plan for storage:

```bash
uv run alt-db entry add --type daily_plan --status posted \
  --title "Daily Plan <YYYY-MM-DD>" \
  --content "<summary_text>\n\n---\n\n<full_plan_text>" \
  --metadata '{"source": "local", "thread_id": "<thread_id>"}'
```
