---
name: weekly-plan
description: Use when starting the week and need to set goals — reviews calendar, issues, routines, and last week's progress
---

# Weekly Planning

## When invoked

### Phase 1: Gather Information

First, determine today's date in JST (do NOT rely on the system-provided `currentDate` which may be UTC):
```bash
TZ=Asia/Tokyo date +%Y-%m-%d
```
Use this date to compute `<monday>` (start of current week) and `<next-monday>` for all subsequent queries.

1. **Google Calendar (full week):**
   Fetch from ALL calendars in parallel. First get the calendar list, then query each:
   ```bash
   gws calendar calendarList list --format yaml
   ```
   Then for each calendar (skip "Holidays in Japan" and "Weather"):
   ```bash
   gws calendar events list --params '{"calendarId": "<calendar-id>", "timeMin": "<monday>T00:00:00+09:00", "timeMax": "<next-monday>T00:00:00+09:00", "singleEvents": true, "orderBy": "startTime"}' --format yaml
   ```
   Read `alt.toml` for calendar context interpretation rules.

2. **GitHub Issues & Milestones:**
   ```bash
   gh issue list --repo <repo> --state open --json number,title,labels,milestone,updatedAt
   ```

3. **All Routines with Status:**
   Run routines skill logic to see what's due this week.

4. **Previous Week's Discord Notes:**
   Review last week's daily report threads for completed/incomplete items.

5. **Knowledge Store Review:**
   - All active goals: `uv run alt-db entry list --type goal --status active`
   - Entries added last week: `uv run alt-db entry list --since 7d`
   - Stale goals (no update in 30+ days): `uv run alt-db entry list --type goal --status active --since 30d` (check for goals NOT in results — those are stale)

### Phase 2: Present Week Overview

```
## Week of YYYY-MM-DD

### Schedule Overview
| Day | Key Events | Available Time |
|-----|-----------|----------------|
| Mon | ... | ~4h |
| ... | ... | ... |

### Development Priorities
- [ ] Issue #123: ...
- [ ] Issue #456: ...

### Routines Due This Week
- Mon: Wash sheets, Clean toilet
- Wed: ...
- Sat: ...

### Goals Overview
- Active goals with current status
- Stale goals (no progress in 30+ days)
- New entries from last week

### Carryover from Last Week
- Items not completed last week
```

### Phase 3: Interactive Goal Setting

Discuss with the user:
- Top 3 goals for this week
- Any deadlines or commitments
- Routine batching strategy (which day for which routines)
- Blockers or concerns

### Phase 4: Output

Based on the discussion, offer output options:
1. **Post to Discord** — Post the weekly plan to the daily channel
2. **Save locally** — Write the plan to a local file
3. **Both**

For Discord posting, read `daily_channel_id` from `alt.toml` and use:
```bash
uv run alt-discord post <daily_channel_id> "<plan text>"
```
Discord has a 2000 character limit per message. If the plan exceeds this, split into multiple messages at natural section boundaries.

After posting to Discord, save the plan to the entries table for Cloud skip detection:
```bash
uv run alt-db entry add --type weekly_plan --status posted \
  --title "Weekly Plan <monday YYYY-MM-DD>" \
  --content "<plan_text>"
```
