# Daily Plan Threading Design Spec

## Overview

Add threading to daily plan Discord posts. The plan remains visible in the channel, and a thread is created on the message so that memos and follow-up notes can be posted throughout the day within the thread.

## Approach

**Message Thread** — use Discord API's `Start Thread from Message` endpoint.

1. Post the plan to the channel using the existing `post_message()`
2. Create a thread on that posted message
3. If the plan exceeds 2000 chars and is split, create the thread on the first chunk and post remaining chunks inside the thread

## Changes

### 1. `src/alt_discord/poster.py` — `create_thread_from_message()`

New function using Discord API `POST /channels/{channel_id}/messages/{message_id}/threads`.

```python
def create_thread_from_message(channel_id: str, message_id: str, name: str) -> dict:
```

- `name`: Thread name (e.g., `📋 2026-04-12 (Sat) Daily Plan`)
- `auto_archive_duration`: 1440 (24 hours)
- Returns: Thread object with `id` and `name`

### 2. `src/alt_discord/cli.py` — `post-thread` command

New CLI subcommand:

```
uv run alt-discord post-thread <channel_id> <thread_name> <message>
```

Flow:
1. `post_message(channel_id, message)` → get first message_id
2. `create_thread_from_message(channel_id, message_id, thread_name)` → get thread_id
3. If message was split (>2000 chars), post remaining chunks to the thread via `post_message(thread_id, chunk)`
4. Output JSON: `{"message_id": "...", "thread_id": "...", "chunks": N}`

### 3. Skill: `daily-plan-cloud` (Phase 4)

Replace plain post with thread-creating post:

**Before:**
```bash
uv run alt-discord post <daily_channel_id> "<plan_text>"
```

**After:**
```bash
uv run alt-discord post-thread <daily_channel_id> "📋 <YYYY-MM-DD> (<Day>) Daily Plan" "<plan_text>"
```

Update entry metadata to include `thread_id`:
```bash
uv run alt-db entry add --type daily_plan --status posted \
  --title "Daily Plan <YYYY-MM-DD>" \
  --content "<plan_text>" \
  --tags '["daily-plan"]' \
  --metadata '{"source": "cloud", "thread_id": "<thread_id>"}'
```

### 4. Skill: `daily-plan` (Phase 4)

Replace webhook-based posting (`curl` to webhook URL) with `alt-discord post-thread`. Same thread naming and metadata pattern as the cloud skill. `source` remains `"local"`.

### 5. Thread Name Format

```
📋 YYYY-MM-DD (Day) Daily Plan
```

Examples:
- `📋 2026-04-12 (Sat) Daily Plan`
- `📋 2026-04-14 (Mon) Daily Plan`

### 6. Tests

- `test_discord_threads.py`: Add `TestCreateThreadFromMessage` — verify correct API endpoint (`/channels/{id}/messages/{msg_id}/threads`) and payload
- `test_discord_cli.py`: Add test for `post-thread` command — verify it calls `post_message` then `create_thread_from_message`, and outputs correct JSON

## Out of Scope

- Thread content management (what to post in threads) — users post manually via Discord or via other skills
- Thread auto-archiving configuration beyond 24h default
- Weekly plan threading (can be added later following the same pattern)
