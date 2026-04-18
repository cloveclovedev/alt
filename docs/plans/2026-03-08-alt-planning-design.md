# alt Planning Feature Design

## Overview

alt is a "second brain" project that uses Claude Code skills for interactive daily planning, routine management, and health tracking. Users converse with Claude Code to build plans, and outputs go to git or Discord.

## Architecture

- **Runtime**: Claude Code skills (invoked interactively via `/daily-plan` etc.)
- **Data definitions**: YAML (git-managed, source of truth)
- **History data**: SQLite `alt.db` (gitignored, local only)
- **Configuration**: TOML `alt.toml` (git-managed)
- **External integrations**: `gws` (Google Calendar), `gh` (GitHub Issues), Discord bot (daily report read/write)

## Directory Structure

```
alt/
├── .claude/
│   └── skills/
│       ├── planning/
│       │   ├── daily-plan.md      # Daily planning skill
│       │   └── weekly-plan.md     # Weekly planning skill
│       ├── routines/
│       │   └── manage-routines.md # Routine check & completion
│       └── health/
│           └── weight-log.md      # Weight logging (future)
├── data/
│   ├── routines/
│   │   ├── household.yml          # Household routine definitions
│   │   ├── dog.yml                # Dog-related routine definitions
│   │   └── health.yml             # Health routine definitions
│   └── alt.db                     # Completion history (gitignored)
├── alt.toml                       # Project configuration
├── .gitignore
├── CLAUDE.md
└── README.md
```

## Data Flow (Daily Planning)

1. User invokes `/daily-plan` skill
2. Information gathering (parallel):
   - `gws calendar events list` -> today's and this week's events
   - `gh issue list` -> development tasks
   - Routine YAML + SQLite history -> overdue and due-today routines
   - Discord daily report channel -> recent progress notes (via anisecord bot)
3. Interactive discussion with Claude Code to decide today's plan
4. Output: post to Discord daily report thread and/or commit to git

## Routine YAML Schema

```yaml
# data/routines/household.yml
- name: Wash the bed and pillow sheets
  interval_days: 7
- name: Clean the toilet
  interval_days: 14
- name: Clean the air conditioner filter
  interval_days: 30
- name: Change tooth brush
  interval_days: 30
- name: Change the outer shaver blade
  interval_days: 120
```

## SQLite Schema (alt.db)

```sql
CREATE TABLE routine_completions (
  id INTEGER PRIMARY KEY,
  routine_name TEXT NOT NULL,
  category TEXT NOT NULL,       -- household, dog, health
  completed_at TEXT NOT NULL,   -- ISO8601
  note TEXT                     -- optional memo
);
```

## Configuration (alt.toml)

```toml
[discord]
bot_token_env = "DISCORD_BOT_TOKEN"
daily_channel_id = "123456789"
webhook_url_env = "DISCORD_WEBHOOK_URL"

[github]
repos = ["mkuri/alt", "mkuri/anisecord"]

[calendar]
timezone = "Asia/Tokyo"
```

## Scope (Phase 1: Planning)

- daily-plan skill
- Routine YAML definitions + SQLite history
- gws / gh / Discord integration
- alt.toml configuration

## Out of Scope (Future)

- Health tracking (weight, meals, exercise)
- News and paper collection
- SNS posting (note, zenn, X)
- Automated schedule execution

## Migration from anisecord

anisecord remains as-is. alt gradually takes over planning functionality:
- TickTick -> YAML routine definitions
- Google Calendar API -> `gws` CLI
- GitHub API -> `gh` CLI
- LLM generation -> Claude Code interactive dialogue
