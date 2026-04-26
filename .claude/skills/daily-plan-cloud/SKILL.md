---
name: daily-plan-cloud
description: "[Cloud scheduled task] Generate and post daily plan if not already posted by the user"
---

# Daily Plan (Cloud Fallback)

Autonomous skill for Cloud scheduled tasks. Checks if a daily plan has been posted today; if not, gathers information and posts one automatically.

## When invoked

### Phase 0: Environment

Install dependencies:
```bash
uv sync
```

### Phase 1: Setup

Determine the current date/time in JST:
```bash
TZ=Asia/Tokyo date '+%Y-%m-%dT%H:%M:%S+09:00'
```
Use this to derive `<today>` (YYYY-MM-DD) and `<sunday>` (end of current week).

Check if a daily plan has already been posted today:
```bash
uv run alt-db entry list --type daily_plan --since 1d
```
If any entry's title contains today's date string (e.g., "Daily Plan 2026-04-09"), end the session silently — no Discord post, no further action.

Read `alt.toml` for configuration:
- `[github] repos` — list of repos to check
- `[discord] daily_channel_id` — plan posting channel
- `[calendar]` — timezone and context interpretation rules

### Phase 2: Data Collection

Run these in parallel:

1. **Google Calendar (today + rest of week):**
   Use Google Calendar MCP connector tools:
   - `list_calendars` to get all calendars
   - `list_events` for each calendar (skip "Holidays in Japan" and "Weather"), timeMin=`<today>T00:00:00+09:00`, timeMax=`<sunday>T23:59:59+09:00`
   - Apply calendar context rules from `alt.toml [calendar]`

   **Calendar notes:**
   - The "Event" calendar is a memo/reminder calendar for optional activities (e.g., basketball open gym, movie discount days). Do not include these in the main schedule — list them separately as optional items in "Rest of Week Overview" or mention as a brief reminder when relevant to today.

2. **GitHub Issues:**
   For each repo in `[github] repos`:
   ```bash
   gh issue list --repo <repo> --state open --json number,title,labels,milestone,updatedAt
   ```

3. **Overdue Routines:**
   Read all YAML files in `data/routines/` to get routine definitions (name, interval_days, active_months, available_days, notes).
   ```bash
   uv run alt-db --json entry list --type routine_event
   ```
   Deduplicate by `title` keeping the latest per routine name. Compare `last_completed + interval_days` against today. Apply active_months and available_days filters per the routines skill logic.

4. **Discord Recent Notes:**
   ```bash
   uv run alt-discord read <daily_channel_id> --after <yesterday_start_iso>
   ```

5. **Knowledge Store Context:**
   ```bash
   uv run alt-db entry list --type goal --status active
   uv run alt-db entry list --since 7d
   uv run alt-db entry list --type goal --due-within 7d
   ```

### Phase 3: Plan Generation

Generate a daily plan autonomously using the collected data. Use this output format:

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

## Rest of Week Overview
- Key events and deadlines
- Optional activities from Event calendar
```

Prioritization logic (autonomous, no user input):
- P0/P1 issues are top priority
- Calendar commitments are immovable
- Overdue routines are flagged
- Suggest time blocking based on calendar gaps

### Phase 4: Save and Post

Post the plan to Discord with threading:
```bash
uv run alt-discord post-thread <daily_channel_id> "📋 <YYYY-MM-DD> (<Day>) Daily Plan" "<plan_text>"
```
Parse the JSON output to extract `thread_id`.

Save the plan to the entries table (include thread_id in metadata):
```bash
uv run alt-db entry add --type daily_plan --status posted \
  --title "Daily Plan <YYYY-MM-DD>" \
  --content "<plan_text>" \
  --metadata '{"source": "cloud", "thread_id": "<thread_id>"}'
```

The `post-thread` command handles the 2000 character limit automatically — the first chunk is posted to the channel, a thread is created on it, and any remaining chunks are posted inside the thread.

### Environment Variables (set in Cloud environment)

- `DISCORD_BOT_TOKEN` — Discord bot token
- `GH_TOKEN` — GitHub Personal Access Token (repo read)
- `NEON_HOST` — Neon PostgreSQL host
- `NEON_DATABASE` — Neon database name
- `NEON_USER` — Neon user
- `NEON_PASSWORD` — Neon password
