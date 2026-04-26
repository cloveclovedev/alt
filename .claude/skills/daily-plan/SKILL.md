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
   Fetch from ALL calendars in parallel. First get the calendar list, then query each:
   ```bash
   # Get all calendar IDs
   gws calendar calendarList list --format yaml
   ```
   Then for each calendar (skip "Holidays in Japan" and "Weather"):
   ```bash
   gws calendar events list --params '{"calendarId": "<calendar-id>", "timeMin": "<today>T00:00:00+09:00", "timeMax": "<sunday>T23:59:59+09:00", "singleEvents": true, "orderBy": "startTime"}' --format yaml
   ```
   Read `alt.toml` for calendar context interpretation rules.

   **Calendar notes:**
   - The "Event" calendar is a memo/reminder calendar for optional activities (e.g., basketball open gym, movie discount days). Do not include these in the main schedule — list them separately as optional items in "Rest of Week Overview" or mention as a brief reminder when relevant to today.

2. **GitHub Issues:**
   Read `alt.toml` [github] repos list. For each repo:
   ```bash
   gh issue list --repo <repo> --state open --json number,title,labels,milestone,updatedAt
   ```

3. **Overdue Routines:**
   Run the routines skill logic (read YAMLs + `uv run alt-db --json entry list --type routine_event`, deduplicate by title keeping latest per routine) to identify overdue and due-soon routines.

4. **Discord Recent Notes (optional):**
   If anisecord bot is accessible, check recent messages in the daily report channel for context from previous days.

5. **Knowledge Store Context:**
   Gather relevant entries from the knowledge store:
   - Active goals: `uv run alt-db entry list --type goal --status active`
   - Recent memos (last 7 days): `uv run alt-db entry list --since 7d`
   - Goals with approaching deadlines: `uv run alt-db entry list --type goal --due-within 7d`

### Phase 2: Present Summary

Present all gathered information organized as:

```
## Today's Schedule (YYYY-MM-DD Day)
- HH:MM-HH:MM [Calendar] Event name
- ...

## Development Tasks (GitHub)
Group issues by priority label (P1 first, then P2). Issues without priority labels are listed separately.
- repo#123: Issue title [label1, label2]
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

### Phase 4: Post to Discord

After the planning discussion is complete, **always** post the finalized plan to Discord. Do not ask — just post it.

Post the plan with threading:
```bash
uv run alt-discord post-thread <daily_channel_id> "📋 <YYYY-MM-DD> (<Day>) Daily Plan" "<plan_text>"
```
Read `daily_channel_id` from `alt.toml` `[discord]` section. Parse the JSON output to extract `thread_id`.

The posted plan should be a concise summary reflecting the discussion outcome (revised schedule, priorities, notes), not the raw Phase 2 output.

The `post-thread` command handles the 2000 character limit automatically — the first chunk is posted to the channel, a thread is created on it, and any remaining chunks are posted inside the thread.

After posting to Discord, save the plan to the entries table for Cloud skip detection:
```bash
uv run alt-db entry add --type daily_plan --status posted \
  --title "Daily Plan <YYYY-MM-DD>" \
  --content "<plan_text>" \
  --metadata '{"source": "local", "thread_id": "<thread_id>"}'
```
