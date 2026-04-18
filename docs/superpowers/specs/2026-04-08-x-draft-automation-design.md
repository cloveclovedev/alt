# X Post Draft Automation — Design Spec

Automated X post draft generation from development activity, integrated into the alt second-brain project.

## Background

As a solo developer, consistent promotional activity is critical for product awareness but difficult to sustain manually. This system automates collection of development activity and generation of X post drafts, posted to Discord for review.

Related: #36 (feature request), PR #18 (original implementation, pre-Neon). This spec supersedes the design in PR #18.

### Goals

- Automate X post draft generation from development activity (git, Discord memos)
- Run autonomously via Cloud scheduled tasks (twice daily)
- Maintain calm, casual engineering tone per tone guide
- Store drafts in existing entries table for dashboard visibility

### Phased Approach

- **Phase 1 (this spec):** Cloud scheduled task -> data collection -> draft generation -> Discord posting
- **Phase 2 (#40):** Dashboard approval flow + X API posting

### Non-Goals (Phase 1)

- X API posting
- Approval/edit flow
- note article idea generation

## Architecture Overview

```
Cloud scheduled task (11:30, 18:30 JST)
    |
    v
x-draft skill execution
    |-- alt-discord: read Discord memos since last draft
    |-- gh API: fetch recent commits, merged PRs, closed issues
    |-- alt-db: get last draft timestamp from entries
    |-- Claude: generate 1-2 draft options (tone-guide)
    |-- alt-db: save drafts as entries (type="x_draft")
    +-- alt-discord: post drafts to journal channel
```

Skill-driven approach: Claude (via the skill) orchestrates the pipeline. Python packages handle I/O only.

## Data Model

Uses existing `entries` table with no schema changes.

```
entries (existing)
  type:     "x_draft"
  title:    Draft summary (one line)
  content:  Draft body text
  status:   "draft" | "approved" | "posted" | "skipped"
  tags:     ["peppercheck", "alt", ...] (related projects)
  metadata: {
    "source_commits": ["repo:hash", ...],
    "source_memo_count": 3,
    "generated_at": "2026-04-08T11:30:00+09:00",
    "tweet_id": null           # Phase 2
  }
```

Last draft time is derived from: `alt-db entry list --type x_draft --json` -> most recent `created_at`.

## Components

### 1. alt_discord package (`src/alt_discord/`)

New Python package alongside existing `alt_db`. Uses stdlib `urllib` only (no extra dependencies).

```
src/alt_discord/
  __init__.py
  reader.py    -- Channel message reading
  poster.py    -- Channel message posting
  cli.py       -- CLI entry point
```

**reader.py:**
- `fetch_messages(channel_id, after_timestamp=None) -> list[dict]` — Fetch up to 100 messages, including thread messages
- `format_messages(messages) -> str` — Format as readable text, sorted by timestamp
- `timestamp_to_snowflake(iso_timestamp) -> str` — Convert ISO 8601 to Discord snowflake ID
- Auth via `DISCORD_BOT_TOKEN` environment variable

**poster.py:**
- `post_message(channel_id, content) -> dict` — Post via Bot API
- `split_message(text) -> list[str]` — Auto-split at 2000 char limit
- Auth via `DISCORD_BOT_TOKEN` environment variable

**cli.py:**
```
alt-discord read <channel_id> [--after <iso_timestamp>]
alt-discord post <channel_id> <message>
```

Logic ported from PR #18's `discord_read.py` and `discord_post.py`.

### 2. x-draft skill (`.claude/skills/x-draft/SKILL.md`)

Cloud scheduled task skill. Execution flow:

1. Get current time in JST
2. Get last draft time: `alt-db entry list --type x_draft --json` -> latest `created_at`. Fallback to 24h ago if none.
3. Data collection (parallel):
   - Discord memos: `alt-discord read <memo_channel_id> --after <last_time>` (from `[discord.content] memo_channel_id`)
   - Discord daily plan: `alt-discord read <daily_channel_id> --after <today_start>` (from existing `[discord] daily_channel_id`)
   - GitHub activity: `gh api` for commits, merged PRs, closed issues per repo in `[github] repos`
4. Evaluate: if nothing noteworthy -> post skip notice to journal channel -> end
5. Read `data/content/tone-guide.md` for style guidelines
6. Generate 1-2 draft options
7. Save each draft: `alt-db entry add --type x_draft --status draft --title "..." --content "..." --tags '[...]' --metadata '{...}'`
8. Post to journal channel: `alt-discord post <draft_channel_id> "..."`

### 3. Tone guide (`data/content/tone-guide.md`)

Carried over from PR #18. Defines calm, casual engineering tone for drafts. Written in Japanese.

## Configuration

### alt.toml changes

```toml
[x]
post_times = ["11:30", "18:30"]

[discord.content]
memo_channel_id = "<actual_id>"
draft_channel_id = "<actual_id>"
```

Non-secret config lives in alt.toml (committed to repo). Channel IDs are not sensitive — they are structural metadata, useless without a bot token.

### pyproject.toml changes

Rename package from `alt-db` to `alt`. Add `alt_discord` to packages and entry points.

```toml
[project]
name = "alt"

[project.scripts]
alt-db = "alt_db.cli:main"
alt-discord = "alt_discord.cli:main"

[tool.hatch.build.targets.wheel]
packages = ["src/alt_db", "src/alt_discord"]
```

### Environment variables (Cloud only)

- `DISCORD_BOT_TOKEN` — Discord bot authentication
- `DATABASE_URL` — Neon PostgreSQL connection string

### Cloud scheduled task setup

| Setting | Value |
|---------|-------|
| Repo | mkuri/alt |
| Network | Full |
| Schedule | Daily at 11:30 and 18:30 JST |
| Prompt | `/x-draft` |
| Setup script | `curl -LsSf https://astral.sh/uv/install.sh \| sh && uv sync && apt install -y gh` |
| Env vars | `DISCORD_BOT_TOKEN`, `DATABASE_URL` |

## Prerequisites (User Action Required)

1. **Discord Bot** — Create bot at Discord Developer Portal, enable MESSAGE_CONTENT intent, grant read + send permissions, invite to server, save token
2. **alt.toml** — Fill in actual channel IDs for `[discord.content]`
3. **Cloud environment** — Create at claude.ai/code with Full network, env vars, setup script, schedule

## Testing

- Unit tests for alt_discord (ported from PR #18): snowflake conversion, message formatting, message splitting
- `uv run pytest` for all tests
- Manual verification: Cloud "Run now" -> verify draft appears in Discord journal channel and in entries table

## File Structure (final)

```
alt/
  .claude/skills/x-draft/SKILL.md
  data/content/tone-guide.md
  src/
    alt_db/         (existing, unchanged)
    alt_discord/
      __init__.py
      reader.py
      poster.py
      cli.py
  tests/
    test_discord_read.py
    test_discord_post.py
  alt.toml          (updated)
  pyproject.toml    (updated)
```
