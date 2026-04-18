# alt — Second Brain powered by Claude Code

Personal planning, routines, and knowledge hub built as a collection of Claude Code skills and CLI tools.

alt turns Claude Code into a personal assistant that:

- **Plans your day and week** from Google Calendar, GitHub issues, and routines
- **Tracks nutrition and body metrics** with Discord-based logging
- **Manages recurring tasks** with smart scheduling and seasonal awareness
- **Drafts and schedules X (Twitter) posts** from your development activity

## Quick Start

1. Clone and install:
   ```bash
   git clone https://github.com/cloveclovedev/alt.git
   cd alt
   uv sync
   ```

2. Configure:
   ```bash
   cp alt.toml.example alt.toml   # Edit with your values
   cp .env.example .env           # Add your credentials
   ```

3. Set up the database:
   ```bash
   # Install Atlas CLI: https://atlasgo.io/getting-started
   atlas migrate apply --url "$DATABASE_URL" --dir "file://db/migrations"
   ```

4. Deploy the webapp (optional but recommended):
   ```bash
   cd webapp
   npm install
   # Deploy to Vercel (recommended) or your preferred platform
   ```

5. Start planning:
   ```
   /daily-plan
   ```

## Skills

| Skill | Type | Description |
|-------|------|-------------|
| `/daily-plan` | Core | Daily planning with calendar, issues, and routines |
| `/weekly-plan` | Core | Weekly review and goal setting |
| `/routines` | Core | Track and manage recurring tasks |
| `/nutrition-check` | Extension | Nutrition tracking from Discord meal posts |
| `/wake-check` | Extension | Wake-up and bedtime alerts via Home Assistant |
| `/x-draft` | Extension | Auto-generate X post drafts from dev activity |
| `/x-post` | Extension | Post approved drafts to X on schedule |

Core skills work with minimal setup (calendar + GitHub + Discord). Extension skills require additional configuration.

## CLI Tools

| Command | Purpose |
|---------|---------|
| `alt-db` | Database CRUD for entries, routines, nutrition, body metrics |
| `alt-discord` | Read and post Discord messages |
| `alt-body` | Import InBody CSV data and calculate fitness metrics |
| `alt-home-assistant` | Home Assistant TTS and device control |

## Architecture

```
┌─────────────────────────────────────┐
│         Claude Code Skills          │
│  (daily-plan, routines, x-draft...) │
└──────────────┬──────────────────────┘
               │ invoke
┌──────────────▼──────────────────────┐
│           CLI Tools                 │
│  (alt-db, alt-discord, alt-body,   │
│   alt-home-assistant)              │
└──┬───────┬───────┬───────┬─────────┘
   │       │       │       │
   ▼       ▼       ▼       ▼
 Neon   Discord  Home    Google
 DB     Bot      Asst.   Calendar
```

## Requirements

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (with Cloud for autonomous skills)
- Python 3.12+ / [uv](https://docs.astral.sh/uv/)
- [Neon PostgreSQL](https://neon.tech/) (free tier works)
- [Vercel](https://vercel.com/) for webapp deployment (other platforms may work but are not officially supported)
- Discord bot token (for posting plans and drafts)
- Google Workspace CLI (`gws`) for calendar access

## Configuration

- `alt.toml` — App settings (Discord channels, GitHub repos, schedule preferences). See `alt.toml.example`.
- `.env` — Credentials (database, Discord bot token). See `.env.example`.

## License

[MIT](LICENSE)
