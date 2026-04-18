# alt — Second Brain

Personal planning and knowledge hub powered by Claude Code skills.

## Project Structure

- `.claude/skills/` — Claude Code skills for planning, routines, health
- Routine definitions are stored in the `routines` DB table (managed via `uv run alt-db routine`)
- `alt.toml` — Project configuration

## Key Commands

- `/daily-plan` — Run daily planning workflow
- `/weekly-plan` — Run weekly planning workflow
- `/routines` — Check and manage routines

## External Tools

- `gws calendar events list` — Google Calendar events
- `gh issue list` — GitHub issues
- Discord — Daily reports via anisecord bot

## Configuration

Edit `alt.toml` for Discord channel IDs, GitHub repos, calendar settings.
Routine definitions are in the `routines` DB table.
