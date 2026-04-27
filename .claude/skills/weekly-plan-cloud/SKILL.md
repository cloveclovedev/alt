---
name: weekly-plan-cloud
description: "[Cloud scheduled task] Generate and post weekly plan if not already posted by the user"
---

# Weekly Plan (Cloud Fallback)

Autonomous skill for Cloud scheduled tasks. Checks if a weekly plan has been posted this week; if not, gathers information and posts one automatically.

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
Use this to compute `<monday>` (start of current week, Monday) and `<next-monday>`.

Check if a weekly plan has already been posted this week:
```bash
uv run alt-db entry list --type weekly_plan --since 7d
```
If any entry's title contains this week's Monday date string (e.g., "Weekly Plan 2026-04-06"), end the session silently — no Discord post, no further action.

Read configuration via:
- `uv run alt-db config get plan.github.repos` — list of repos to check
- `uv run alt-db config get plan.discord.channel_id` — plan posting channel
- `uv run alt-db config get plan.google_calendar.context` — calendar interpretation rules

### Phase 2: Data Collection

Run these in parallel:

1. **Google Calendar (full week):**
   Use Google Calendar MCP connector tools:
   - `list_calendars` to get all calendars
   - `list_events` for each calendar (skip "Holidays in Japan" and "Weather"), timeMin=`<monday>T00:00:00+09:00`, timeMax=`<next-monday>T00:00:00+09:00`
   - Apply calendar context rules from the `plan.google_calendar.context` value

   **Calendar notes:**
   - The "Event" calendar is a memo/reminder calendar for optional activities (e.g., basketball open gym, movie discount days). Do not include these in the main schedule — list them separately as optional items or mention as brief reminders.

2. **GitHub Issues & Milestones:**
   For each repo in `plan.github.repos`:
   ```bash
   gh issue list --repo <repo> --state open --json number,title,labels,milestone,updatedAt
   ```

3. **Routines (week view):**
   ```bash
   uv run alt-db --json config get routines
   uv run alt-db --json entry list --type routine_event
   ```
   `config.routines` is a JSON object keyed by routine name; each value has: `content` (optional notes), `category`, `interval_days`, `active_months` (optional), `available_days` (optional).
   For routine_event entries, deduplicate by `title` keeping the latest per routine name. Determine which routines will be due this week based on last_completed + interval_days. Apply active_months filter.

4. **Last week's daily plans:**
   ```bash
   uv run alt-db entry list --type daily_plan --since 7d
   ```

5. **Knowledge Store:**
   ```bash
   uv run alt-db entry list --type goal --status active
   uv run alt-db entry list --since 7d
   ```
   Check for stale goals: active goals NOT updated in 30+ days (those not appearing in `--since 30d` results).

### Phase 3: Plan Generation

Generate a weekly plan autonomously using the collected data. Use this output format:

```
## Week of YYYY-MM-DD

### Schedule Overview
| Day | Key Events | Available Time |
|-----|-----------|----------------|
| Mon | ... | ~4h |
| Tue | ... | ~6h |
| ... | ... | ... |

### Development Priorities
- [ ] repo#123: Issue title [P1]
- [ ] repo#456: Issue title [P2]

### Routines Due This Week
- Mon: Clean toilet, Change toothbrush
- Wed: Wash sheets
- Sat: ...

### Goals Overview
- Active goals with current status
- Stale goals (no progress in 30+ days) — flagged
- New entries from last week

### Carryover from Last Week
- Items from last week's daily plans not completed
```

Prioritization logic (autonomous, no user input):
- P0/P1 issues are top priority for the week
- Distribute routines across available days based on `available_days` field
- Flag stale goals for attention
- Suggest top 3 goals for the week based on priority and deadlines

### Phase 4: Save and Post

Save the plan to the entries table:
```bash
uv run alt-db entry add --type weekly_plan --status posted \
  --title "Weekly Plan <monday YYYY-MM-DD>" \
  --content "<plan_text>"
```

Post the plan to Discord:
```bash
uv run alt-discord post <daily_channel_id> "<plan_text>"
```
Discord has a 2000 character limit per message. If the plan exceeds this, split into multiple messages at natural section boundaries.

### Environment Variables (set in Cloud environment)

- `DISCORD_BOT_TOKEN` — Discord bot token
- `GH_TOKEN` — GitHub Personal Access Token (repo read)
- `NEON_HOST` — Neon PostgreSQL host
- `NEON_DATABASE` — Neon database name
- `NEON_USER` — Neon user
- `NEON_PASSWORD` — Neon password
