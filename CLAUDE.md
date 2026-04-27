# alt — Second Brain

Personal planning and knowledge hub powered by Claude Code skills.

## Project Structure

- `.claude/skills/` — Claude Code skills for planning, routines, health
- Routine definitions live in `config.routines` (one JSON object keyed by routine title); completion events are entries (type `routine_event`)
- `config` table — Project configuration (managed via `uv run alt-db config`)

## Key Commands

- `/daily-plan` — Run daily planning workflow
- `/weekly-plan` — Run weekly planning workflow
- `/routines` — Check and manage routines

## External Tools

- `gws calendar events list` — Google Calendar events
- `gh issue list` — GitHub issues
- Discord — Daily reports via bot

## Configuration

Manage Discord channel IDs, GitHub repos, calendar settings via `uv run alt-db config set/get`.
Routine definitions are managed via `uv run alt-db config get/set routines` (single JSON object keyed by routine title).
