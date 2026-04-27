---
name: wake-check-cloud
description: "[Cloud scheduled task] Check if user is awake and trigger Google Home TTS if not"
---

# Wake Check (Cloud)

Autonomous skill for Cloud scheduled tasks. Checks if the user has woken up (by looking for activity signals) and sends a Google Home TTS wake-up message if not. Also handles late-night bedtime warnings.

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
Use this to derive `<today>` (YYYY-MM-DD), `<current_time>` (HH:MM), and `<day_of_week>`.

Read configuration via:
- `uv run alt-db config get wake.wakeup_time`
- `uv run alt-db config get wake.google_calendar.adaptive`
- `uv run alt-db config get wake.prep_minutes`
- `uv run alt-db config get wake.escalation.interval_minutes`
- `uv run alt-db config get wake.escalation.max_attempts`
- `uv run alt-db config get wake.night.bedtime`
- `uv run alt-db config get wake.night.google_calendar.lookahead`
- `uv run alt-db config get core.home_assistant.tts_entity`
- `uv run alt-db config get plan.discord.channel_id`
- `uv run alt-db config get core.timezone`
- `uv run alt-db config get plan.google_calendar.context`

Determine the **active scenario** based on current time:

1. **Morning wake-up**: from `default_wakeup_time` to `wakeup_time + (max_attempts * interval_minutes)`
   - If `calendar_adaptive = true`: check Google Calendar for the first event of the day. Compute `event_start - prep_minutes` as the adaptive wake-up time. Use the earlier of adaptive and default.
2. **Late night warning**: from `default_bedtime` to 02:00 next day
   - If `calendar_lookahead = true`: check Google Calendar for next morning's first event. Compute recommended bedtime as `event_start - prep_minutes - target_sleep_hours (7h)`.
3. **No active scenario**: current time is outside all check windows → end session silently.

### Phase 2: Activity and Escalation Check

**For morning wake-up:**

Check if already awake (any of these signals = awake, end session):
```bash
uv run alt-db entry list --type daily_plan --since 1d --json
```
Parse the JSON output. If any entry has `metadata.source == "local"` and its title contains today's date → **user is awake, end session silently**.

Also check Discord for user messages since wake-up time:
```bash
uv run alt-discord read <daily_channel_id> --after <wakeup_time_iso>
```
If messages from the user (not bots) exist → **user is awake, end session silently**.

Check escalation state:
```bash
uv run alt-db entry list --type wake_event --since 1d --json
```
Count entries with `metadata.scenario == "morning"` for today. If count >= `max_attempts` → **max attempts reached, end session silently**.

**For late night warning:**

Check if user is still active:
```bash
uv run alt-discord read <daily_channel_id> --after <bedtime_iso>
```
If no messages after bedtime → **user may already be asleep, end session silently**.

Check escalation state:
```bash
uv run alt-db entry list --type wake_event --since 1d --json
```
Count entries with `metadata.scenario == "night"` for today. If count >= `max_attempts` → **max attempts reached, end session silently**.

### Phase 3: Generate TTS Message

Generate a context-aware message based on:
- Current time
- Scenario (morning or night)
- Attempt number (1st = gentle, 2nd = informational, 3rd = urgent)
- Calendar events (today's schedule for morning, tomorrow's first event for night)

**Morning message guidelines:**
- Attempt 1 (gentle): "おはようございます、{時刻}です"
- Attempt 2 (informational): "{時刻}です。今日は{時刻}から{予定}があります"
- Attempt 3 (urgent): "{時刻}です。そろそろ起きないと間に合いません"

**Night message guidelines:**
- Attempt 1 (advisory): "明日は{時刻}から{予定}があります、そろそろ寝ましょう"
- Attempt 2 (insistent): "もう{時刻}を過ぎました。{N}時間睡眠を確保するなら今です"

Speak in Japanese. Keep messages concise (1-2 sentences). Do not use robotic/template-like phrasing — speak naturally.

### Phase 4: Send TTS and Record

Send the TTS message:
```bash
uv run alt-home-assistant tts "<message>" --entity <tts_entity>
```

Record the attempt in the database:
```bash
uv run alt-db entry add --type wake_event --status sent \
  --title "Wake: <scenario> attempt <N>" \
  --content "<message>" \
  --metadata '{"attempt": <N>, "scenario": "<morning|night>", "target_time": "<HH:MM>"}'
```

### Environment Variables (set in Cloud environment)

All existing Cloud task variables plus:

| Variable | Purpose |
|---|---|
| `HA_URL` | Public URL of Home Assistant (via Cloudflare Tunnel) |
| `HA_TOKEN` | Home Assistant Long-Lived Access Token |
| `DISCORD_BOT_TOKEN` | Discord bot token (existing) |
| `GH_TOKEN` | GitHub PAT (existing, for calendar-adaptive) |
| `NEON_HOST` | Neon PostgreSQL host (existing) |
| `NEON_DATABASE` | Neon database name (existing) |
| `NEON_USER` | Neon user (existing) |
| `NEON_PASSWORD` | Neon password (existing) |
