# Daily/Weekly Plan Cloud Automation — Design Spec

Automated fallback for daily and weekly planning via Cloud scheduled tasks. If the user hasn't posted a plan by a set time, Cloud generates and posts one autonomously.

## Background

The existing `/daily-plan` and `/weekly-plan` skills are interactive — they gather information, present it, discuss priorities with the user, and post to Discord. This works well but depends on the user remembering to run them. This spec adds Cloud-based fallback skills that ensure a plan is always posted.

Related: x-draft-cloud (established Cloud skill pattern), #44 (Neon HTTP migration).

### Goals

- Ensure a daily plan is posted every day, even if the user forgets to run `/daily-plan`
- Ensure a weekly plan is posted every week, even if the user forgets to run `/weekly-plan`
- Preserve existing local interactive workflows unchanged (minimal modification)
- Follow the established x-draft-cloud pattern for Cloud skills

### Phased Approach

- **Phase 1 (this spec):** Cloud fallback skills + local DB save addition
- **Phase 2 (#46):** Local skills detect existing Cloud-generated plan and offer to revise it interactively

### Non-Goals (Phase 1)

- Replacing or modifying the interactive local planning flow
- Cloud-generated plan revision from local (#46)
- Notification when Cloud skips (plan already posted)

## Architecture Overview

```
Daily (10:00 JST):
  Cloud daily-plan-cloud starts
    → Check DB for today's daily_plan entry
    → Already posted → skip silently → end
    → Not posted → collect data → generate plan → save to DB → post to Discord

Weekly (Sunday 10:00 JST):
  Cloud weekly-plan-cloud starts
    → Check DB for this week's weekly_plan entry
    → Already posted → skip silently → end
    → Not posted → collect data → generate plan → save to DB → post to Discord

Local /daily-plan (unchanged flow + DB save):
  → Interactive planning as before
  → Phase 4: save to DB (type=daily_plan) + post to Discord
  → Cloud sees DB entry next morning → skips
```

Skill-driven approach: Claude (via the skill) orchestrates the pipeline. Python packages handle I/O only.

## Data Model

Uses existing `entries` table with no schema changes. Two new type values:

| type | title | content | status | tags |
|------|-------|---------|--------|------|
| `daily_plan` | "Daily Plan 2026-04-09" | Plan body text | `posted` | `["daily-plan"]` |
| `weekly_plan` | "Weekly Plan 2026-04-07" | Plan body text | `posted` | `["weekly-plan"]` |

Skip detection: query `alt-db entry list --type daily_plan --since 1d` and check if any entry's title contains today's date string (e.g., "Daily Plan 2026-04-09"). Using the title date avoids 24-hour boundary issues with `--since`. For weekly, check if an entry title contains this week's Monday date.

## Components

### 1. daily-plan-cloud skill (`.claude/skills/daily-plan-cloud/SKILL.md`)

Cloud scheduled task skill. Fully autonomous, no interactive phases.

**Phase 0: Environment**
```bash
uv sync
```

**Phase 1: Setup**
- Determine current date/time in JST
- Check for existing plan today:
  ```bash
  uv run alt-db entry list --type daily_plan --since 1d
  ```
  If an entry title contains today's date → end session silently (no Discord post).
- Read `alt.toml` for configuration (github repos, discord channel IDs, calendar context)

**Phase 2: Data Collection (parallel)**

1. **Google Calendar (today + rest of week):**
   Use Google Calendar MCP connector tools:
   - `list_calendars` to get all calendars
   - `list_events` for each calendar (skip "Holidays in Japan" and "Weather"), timeMin=today, timeMax=end of week
   - Apply calendar context rules from `alt.toml [calendar]`

2. **GitHub Issues:**
   For each repo in `[github] repos`:
   ```bash
   gh issue list --repo <repo> --state open --json number,title,labels,milestone,updatedAt
   ```

3. **Overdue Routines:**
   Read YAML files in `data/routines/` + run `uv run alt-db routine all` to identify overdue and due-soon routines.

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

**Phase 3: Plan Generation**

Generate a daily plan autonomously using the collected data. Apply the same output format as the local daily-plan skill:

```
## Today's Schedule (YYYY-MM-DD Day)
- HH:MM-HH:MM [Calendar] Event name

## Development Tasks (GitHub)
Group by priority label (P1 first, then P2).
- repo#123: Issue title [label1, label2]

## Routines Due
- Overdue: ...
- Due soon: ...

## Goals & Reminders
- Active goals with status
- Goals with approaching deadlines

## Rest of Week Overview
- Key events and deadlines
```

Prioritization logic (autonomous, no user input):
- P0/P1 issues are top priority
- Calendar commitments are immovable
- Overdue routines are flagged
- Time blocking based on calendar gaps

**Phase 4: Save and Post**

Save to DB:
```bash
uv run alt-db entry add --type daily_plan --status posted \
  --title "Daily Plan <YYYY-MM-DD>" \
  --content "<plan_text>" \
  --tags '["daily-plan"]'
```

Post to Discord:
```bash
uv run alt-discord post <daily_channel_id> "<plan_text>"
```
Split at 2000 char limit if needed.

### 2. weekly-plan-cloud skill (`.claude/skills/weekly-plan-cloud/SKILL.md`)

Cloud scheduled task skill. Same pattern as daily-plan-cloud but for weekly scope.

**Phase 0: Environment**
```bash
uv sync
```

**Phase 1: Setup**
- Determine current date/time in JST, compute Monday (start of week) and next Monday
- Check for existing weekly plan:
  ```bash
  uv run alt-db entry list --type weekly_plan --since 7d
  ```
  If an entry title contains this week's Monday date → end session silently.
- Read `alt.toml` for configuration

**Phase 2: Data Collection (parallel)**

1. **Google Calendar (full week):**
   Use Google Calendar MCP connector tools:
   - `list_calendars` → `list_events` per calendar, timeMin=Monday, timeMax=next Monday

2. **GitHub Issues & Milestones:**
   For each repo in `[github] repos`:
   ```bash
   gh issue list --repo <repo> --state open --json number,title,labels,milestone,updatedAt
   ```

3. **Routines (week view):**
   Read YAML files + `uv run alt-db routine all` to determine what's due this week.

4. **Last week's daily plans:**
   ```bash
   uv run alt-db entry list --type daily_plan --since 7d
   ```

5. **Knowledge Store:**
   ```bash
   uv run alt-db entry list --type goal --status active
   uv run alt-db entry list --since 7d
   ```
   Check for stale goals (active goals NOT updated in 30+ days).

**Phase 3: Plan Generation**

Generate weekly plan autonomously:

```
## Week of YYYY-MM-DD

### Schedule Overview
| Day | Key Events | Available Time |
|-----|-----------|----------------|
| Mon | ... | ~4h |

### Development Priorities
- [ ] Issue #123: ...

### Routines Due This Week
- Mon: ...
- Wed: ...

### Goals Overview
- Active goals with status
- Stale goals flagged

### Carryover from Last Week
- Items from last week's daily plans not completed
```

**Phase 4: Save and Post**

Save to DB:
```bash
uv run alt-db entry add --type weekly_plan --status posted \
  --title "Weekly Plan <YYYY-MM-DD>" \
  --content "<plan_text>" \
  --tags '["weekly-plan"]'
```

Post to Discord:
```bash
uv run alt-discord post <daily_channel_id> "<plan_text>"
```

### 3. Local skill modifications (minimal)

**daily-plan:** Add to Phase 4 (after Discord webhook post):
```bash
uv run alt-db entry add --type daily_plan --status posted \
  --title "Daily Plan <YYYY-MM-DD>" \
  --content "<plan_text>" \
  --tags '["daily-plan"]'
```

**weekly-plan:** Add to Phase 4 (after Discord post):
```bash
uv run alt-db entry add --type weekly_plan --status posted \
  --title "Weekly Plan <YYYY-MM-DD>" \
  --content "<plan_text>" \
  --tags '["weekly-plan"]'
```

## Configuration

### alt.toml

No changes needed. Existing configuration covers all required values:
- `[discord] daily_channel_id` — plan posting channel
- `[github] repos` — repos to check
- `[calendar]` — timezone and context rules

### Cloud Scheduled Task Setup

**daily-plan-cloud:**

| Setting | Value |
|---------|-------|
| Repo | mkuri/alt |
| Network | Full |
| Connectors | Google Calendar |
| Schedule | Daily at 10:00 JST |
| Prompt | `/daily-plan-cloud` |
| Setup script | `curl -LsSf https://astral.sh/uv/install.sh \| sh && uv sync && apt install -y gh` |
| Env vars | `DISCORD_BOT_TOKEN`, `GH_TOKEN`, `NEON_HOST`, `NEON_DATABASE`, `NEON_USER`, `NEON_PASSWORD` |

**weekly-plan-cloud:**

| Setting | Value |
|---------|-------|
| Repo | mkuri/alt |
| Network | Full |
| Connectors | Google Calendar |
| Schedule | Every Sunday at 10:00 JST |
| Prompt | `/weekly-plan-cloud` |
| Setup script | Same as daily-plan-cloud |
| Env vars | Same as daily-plan-cloud |

## Scope Summary

### In scope (this PR)
- New skill: `.claude/skills/daily-plan-cloud/SKILL.md`
- New skill: `.claude/skills/weekly-plan-cloud/SKILL.md`
- Modify: `.claude/skills/daily-plan/SKILL.md` (add DB save to Phase 4)
- Modify: `.claude/skills/weekly-plan/SKILL.md` (add DB save to Phase 4)

### Out of scope (Phase 2, separate PR)
- Local skill detecting Cloud-generated plan and offering revision flow (#46)
- Any changes to plan output format

## Testing

- Manual verification: Run Cloud skills via "Run now" → verify plan appears in Discord and entries table
- Verify skip logic: Run locally first, then trigger Cloud → confirm Cloud skips
- Verify skip logic (reverse): Let Cloud post, verify entry exists in DB

## File Structure (changes only)

```
alt/
  .claude/skills/
    daily-plan/SKILL.md          (modified — add DB save)
    weekly-plan/SKILL.md         (modified — add DB save)
    daily-plan-cloud/SKILL.md    (new)
    weekly-plan-cloud/SKILL.md   (new)
```
