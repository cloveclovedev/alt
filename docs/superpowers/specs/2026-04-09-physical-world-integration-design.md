# Physical World Integration — Design Spec

Connect alt to the physical world: detect sleep state and trigger real-world actions (Google Home TTS) to manage wake-up, nap detection, and bedtime enforcement.

## Background

alt currently operates in the digital domain — Discord, GitHub, Google Calendar, Neon DB. This spec extends alt into the physical world by integrating with Home Assistant and sleep sensors, starting with activity-based wake-up management and progressing to hardware sensor integration.

Related: daily-plan-cloud (Cloud skill pattern), alt-discord (CLI tool pattern), alt-body (health data pattern).

### Goals

- Wake the user via Google Home TTS when they oversleep past their target wake-up time
- Detect oversleeping using existing digital activity signals (Phase 1) and sleep sensors (Phase 2)
- Warn the user about staying up too late based on next-day calendar events
- Establish Home Assistant as the bridge between alt and physical devices

### Phased Approach

- **Phase 1 (this spec, primary):** Activity-based detection + Home Assistant + Google Home TTS
- **Phase 2 (future):** Withings Sleep Analyzer integration for sensor-based detection, nap detection, sleep data collection

### Non-Goals (Phase 1)

- Sleep stage analysis or sleep quality tracking
- Daytime nap detection (requires Withings sensor, Phase 2)
- Lighting or appliance control (future expansion via HA)
- Home Assistant automations running independently of alt (alt Cloud Tasks drive all logic)

## Architecture Overview

```
┌─────────────────────────────────────┐
│         alt Cloud Task              │
│  (wake-check-cloud skill)           │
│                                     │
│  1. Current time vs target wake time│
│  2. Activity check                  │
│     - DB: daily_plan (source:local) │
│     - Discord: user messages        │
│  3. [Phase2] Withings API: in bed?  │
│  4. Decision: should we wake?       │
│                                     │
│  5. alt-home-assistant CLI → HA API │
└──────────┬──────────────────────────┘
           │ HTTPS (Cloudflare Tunnel)
┌──────────▼──────────────────────────┐
│    Home Assistant (Raspberry Pi)     │
│  - Google Cast integration           │
│  - REST API (Long-Lived Token)       │
│  - [Phase2] Withings integration     │
│  - [Future] SwitchBot, lighting      │
└──────────┬──────────────────────────┘
           │ LAN (Cast protocol)
      Google Home (TTS output)
```

## Scenarios

### Scenario 1: Morning Wake-Up

The primary use case. A Cloud scheduled task runs at intervals starting from the target wake-up time.

```
06:30  Target wake-up time (from alt.toml or calendar-adaptive)
06:30  wake-check-cloud starts, checks for activity signals
06:35  No signal → Google Home: "おはようございます、6時半ですよ"
06:45  Still no signal → 2nd attempt with calendar context
07:00  Still no signal → 3rd attempt (stronger message)
07:xx  User runs /daily-plan → source:local entry in DB → check passes, done
```

### Scenario 2: Late Night Warning

Checks the next day's Google Calendar and warns the user to go to bed. Activity is detected via Discord messages in the daily channel after bedtime.

```
23:00  Check next-day calendar → morning meeting at 09:00
       → Google Home: "明日は9時から予定があります、そろそろ寝ましょう"
24:00  Discord messages found after 23:00 → user still active → 2nd warning
```

### Scenario 3: Daytime Nap Detection (Phase 2)

Requires Withings Sleep Analyzer. Detects unexpected bed occupancy during daytime hours.

```
Weekday 10:00-22:00: Withings detects bed occupancy
→ 30 minutes elapsed, still in bed
→ Google Home: "昼寝が30分を超えました"
```

## Components

### 1. `alt-home-assistant` CLI Tool (New)

A thin wrapper around the Home Assistant REST API, following the same pattern as `alt-discord` and `alt-db` (urllib-based, no external HTTP dependencies).

```
alt-home-assistant tts "<message>" [--entity media_player.living_room]
alt-home-assistant state <entity_id>
alt-home-assistant call <domain> <service> [--data '{}']
```

**Subcommands:**

| Command | Purpose |
|---|---|
| `tts` | Send TTS message to a Google Home device via HA |
| `state` | Get current state of an HA entity |
| `call` | Call an arbitrary HA service |

**Environment variables:** `HA_URL`, `HA_TOKEN`

**Implementation:** `urllib.request` (standard library only, consistent with `alt-db` and `alt-discord`).

**TTS flow:**
```
alt-home-assistant tts "おはようございます"
  → POST {HA_URL}/api/services/tts/speak
    Headers: Authorization: Bearer {HA_TOKEN}
    Body: {"entity_id": "...", "message": "..."}
  → HA → Google Cast → Google Home speaks
```

### 2. `wake-check-cloud` Skill (New Cloud Task)

A Cloud scheduled task following the established pattern (daily-plan-cloud, x-draft-cloud).

**Execution schedule:** Every 5 minutes during active check windows (morning wake-up, late night).

**Phases:**

1. **Time check** — Is it within a wake/sleep check window?
   - Morning: from `default_wakeup_time` to `wakeup_time + max_attempts * interval`
   - Night: from `default_bedtime` onward
   - Calendar-adaptive: adjust based on next event requiring wake-up

2. **Activity check** — Has the user woken up?
   - Query DB: `alt-db entry list --type daily_plan --since 1d` → filter for `metadata.source == "local"`
   - Query Discord: check for user messages in daily channel since wake-up time
   - [Phase 2] Query Withings API for bed occupancy state

3. **Escalation check** — How many attempts today?
   - Query DB: `alt-db entry list --type wake_event --since 1d`
   - If `max_attempts` reached → skip

4. **Action** — Send TTS via HA
   - Generate context-aware message (Claude generates based on time, calendar, attempt number)
   - `alt-home-assistant tts "<message>"`
   - Record attempt: `alt-db entry add --type wake_event ...`

### 3. Existing Skill Modifications

**`daily-plan/SKILL.md`:** Add `--metadata '{"source": "local"}'` to the `alt-db entry add` command.

**`daily-plan-cloud/SKILL.md`:** Add `--metadata '{"source": "cloud"}'` to the `alt-db entry add` command.

This allows `wake-check-cloud` to distinguish user-initiated plans from automated fallbacks.

## Configuration

### alt.toml Extensions

```toml
[wake]
default_wakeup_time = "06:30"
calendar_adaptive = true        # Adjust wake time based on first calendar event
prep_minutes = 60               # Wake up this many minutes before first event

[wake.escalation]
interval_minutes = 10           # Minutes between TTS attempts
max_attempts = 3                # Maximum TTS attempts per session

[wake.night]
default_bedtime = "23:00"
calendar_lookahead = true       # Check next morning's calendar for bedtime advice

[home_assistant]
url = "http://homeassistant.local:8123"
tts_entity = "media_player.living_room"
# HA_TOKEN provided via environment variable
```

## TTS Messages

Messages are generated by Claude (within the Cloud Task) based on context, not fixed templates.

**Morning escalation pattern:**
1. Gentle: "おはようございます、7時です"
2. Informational: "7時10分です。今日は9時からミーティングがあります"
3. Urgent: "7時20分です。そろそろ起きないと間に合いません"

**Late night pattern:**
1. Advisory: "明日は9時から予定があります、そろそろ寝ましょう"
2. Insistent: "もう0時を過ぎました。6時間睡眠を確保するなら今です"

## Escalation Tracking

Each TTS attempt is recorded in the DB:

```bash
uv run alt-db entry add --type wake_event --status sent \
  --title "Wake attempt 1" \
  --metadata '{"attempt": 1, "scenario": "morning", "message": "..."}'
```

The Cloud Task queries these records to determine the current attempt number and whether to continue.

## Infrastructure

### Home Assistant Setup

1. Install Home Assistant OS on Raspberry Pi (existing hardware)
2. Enable Google Cast integration (auto-discovers Google Home on LAN)
3. Create Long-Lived Access Token for API access
4. Expose via Cloudflare Tunnel for external access

### Cloudflare Tunnel

```
RPi: cloudflared daemon → outbound connection to Cloudflare
Cloudflare: ha.example.com → tunnel → RPi localhost:8123
Claude Cloud Task → HTTPS GET/POST ha.example.com → reaches HA
```

- No router port forwarding required (outbound-only from RPi)
- Free tier is sufficient
- IoT WiFi SSID (WPA2/WPA3 mixed mode) recommended for Withings compatibility

### Cloud Task Environment Variables

In addition to existing variables (DISCORD_BOT_TOKEN, GH_TOKEN, NEON_*):

| Variable | Purpose |
|---|---|
| `HA_URL` | Public URL of Home Assistant (via Cloudflare Tunnel) |
| `HA_TOKEN` | Home Assistant Long-Lived Access Token |

## Phase 2: Withings Integration (Overview)

Detailed design will be created when Withings Sleep Analyzer is acquired. Key points:

### Hardware

- Withings Sleep Analyzer placed under mattress (above metal bed frame, with cardboard for stability)
- WiFi: IoT SSID with WPA2/WPA3 mixed mode (AXE5400 router)

### `alt-withings` CLI Tool

- OAuth2 authentication with Withings API
- Query sleep state (in bed / out of bed) and sleep data
- Store sleep metrics in alt DB for visualization

### Enhanced Detection Logic

```
Phase 1: Activity-based only
  daily_plan (source:local) exists? → awake

Phase 2: Sensor + Activity combined
  Withings: out of bed? AND daily_plan (source:local) exists?
  → Both required for "fully awake"
  → Out of bed but no daily_plan → gentle nudge: "起きてるなら計画を立てましょう"
```

### Daytime Nap Detection

- Monitor Withings bed occupancy during configurable hours (weekday 10:00-22:00)
- Alert after configurable threshold (default: 30 minutes)

### Sleep Data Collection

- Store sleep metrics in alt DB (new table or extend body_measurements)
- Future: webapp visualization on `/sleep` page (out of Phase 2 scope)

### Pet Considerations

- 2kg dog frequently on bed — Withings uses ballistocardiography (heart activity), unlikely to trigger on small pets
- Sensitivity settings available for tuning if needed
