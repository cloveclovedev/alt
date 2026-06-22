# Training Tracking — Design

- Date: 2026-06-21
- Status: Approved (design)

## Summary

alt currently represents training only as recurring Google Calendar `FIT |`
events. Those events are schedule-only: there is no record of whether a session
was actually done, nor of performance progression (working weights, reps, or the
vertical-jump KPI). This document designs a training-tracking subsystem so that
`daily-plan` and `weekly-plan` can review both adherence and progression.

Workouts are logged through a dedicated Discord channel and parsed into
structured entries, mirroring the existing nutrition-logging pattern
(`nutrition-check-cloud`). Review is **pull-first**: no new scheduled cloud
infrastructure is introduced. A push variant (reminders, missing-log detection,
weekly auto-summary) is explicitly deferred.

This addresses the long-standing gap tracked by alt#6 ("feat: add exercise
tracking system").

### Goals

- Track **adherence** (was the weekly base met) and **performance** (key-lift
  numbers and KPI measurements).
- Log workouts via a dedicated Discord channel, parsed into structured entries.
- Surface "today's menu" plus recent numbers in `daily-plan`; surface weekly
  adherence plus progression in `weekly-plan`.
- Reuse existing patterns: the entries table, the `routines` skill structure,
  and the Discord read/parse flow.

### Non-goals (deferred)

- A cloud push skill (`workout-check-cloud`): reminders, missing-log detection,
  weekly auto-summary.
- Webapp progression charts (could later ride on the existing body dashboard).
- Automatic per-exercise 1RM estimation.

## Terminology

To keep usage consistent across the skill, config, entry type, and docs:

- **training** — the domain / program / feature: the ongoing practice and its
  master plan. Used for the skill (`training`), the config key
  (`config.training`), and the review concepts ("training plan", "Training
  Review").
- **workout** — a single training session: one logged activity (one gym visit,
  one HIIT session, one basketball game). This is the unit of record — one
  `workout_log` entry per workout. A workout may be assembled from one or more
  Discord posts (e.g., one post per exercise), which the parser merges into a
  single entry. "Today's workout" is the session planned for today; its **menu**
  is that workout's list of exercises.

## Data Model

### `config.training` (master data)

A single config key (`training`) holding a JSON object. The canonical content is
a markdown string; a small set of machine-readable fields drive deterministic
logic. Managed via `uv run alt-db config get/set training` (same flow as
`routines`).

| Field | Type | Purpose |
|---|---|---|
| `plan` | string (markdown) | Canonical master plan: purpose, places, weekday→menu mapping, irregularity algorithm, safety/etiquette notes. Free-form; the user appends rules here over time. |
| `key_lifts` | string[] | Stable exercise identifiers used to drive progression review. |
| `kpi` | string[] | Performance KPIs measured occasionally (e.g., vertical jump). |
| `weekly_base` | object | Adherence rule as counts, e.g. `{ "personal": 1, "gym_24h_min": 1, "gym_24h_target": 2 }`. |
| `discord.channel_id` | string | Dedicated training channel used for logging. |

**Canonicality.** `plan` (prose) is authoritative. The structured fields are
hints derived from the plan; if they diverge, the prose wins and both should be
updated. All consumers are LLM-driven skills, so they read the prose directly —
no rigid schema is required for the weekday→menu mapping or the irregularity
rules.

**Why config (not a knowledge entry).** This is behavior-driving reference data
for skills (a peer of `routines` and the calendar context), not a reviewable
knowledge memo. Keeping it in config avoids polluting the `weekly-plan` "recent
entries" and webapp `/entries` feeds, and it slots into the per-skill config
webapp UI roadmap.

### `workout_log` entry type

A new entry `type` by convention only. The entries table `type` column is an open
string (see `src/alt_db/entries.py`), so **no schema migration is required** —
this matches how `routine_event`, `nutrition_log`, etc. were introduced.

One entry per workout (one training session).

- `type`: `workout_log`
- `title`: human-readable, date-stamped workout label — e.g.
  `Workout YYYY-MM-DD — 24h Gym Lower`.
- `status`: not used.
- `content`: raw Discord message text (provenance / re-parse source).
- `metadata`:
  - `logged_date` (`YYYY-MM-DD`) — workout date, derived from the message
    timestamp in JST unless the post states otherwise.
  - `place` — one of `personal | gym_24h | home_elevator | floor1 | basketball | other`.
  - `workout_type` — one of `upper | lower | full | hiit | basketball | other`.
  - `exercises` (array) — each item `{ name, weight_kg?, reps?, sets?, value?, unit?, note? }`.
    `weight_kg`/`reps`/`sets` cover resistance work; `value`/`unit` cover
    measurements (e.g., a vertical-jump test of 60 cm) and HIIT
    (rounds/duration). This keeps measurements and HIIT in the same structure
    without introducing additional entry types.
  - `rpe` (number, optional) — overall workout intensity.
  - `note` (string, optional).
  - `source_message_ids` (string[]) — Discord message ids that contributed to
    this workout, used for idempotency. A workout assembled from several posts
    carries all of their ids.
  - `estimated_by` — `discord_parse` (provenance; reserved for future modes).
- `parent`: not used.

**Idempotency & multi-post workouts.** Before processing a message, the parser
runs `entry search "<message_id>"` and skips it if any `workout_log` already
lists that id in `source_message_ids`. When a new message belongs to an existing
workout for the same `logged_date` + `place` (e.g., a follow-up post adding more
exercises), the parser appends its exercises to that entry and adds the message
id to `source_message_ids`, rather than creating a second workout. Otherwise it
creates a new `workout_log`. The rare two-workouts-same-place-same-day case is
disambiguated by a note in the post (e.g., "pm"). This preserves the invariant:
one `workout_log` entry = one workout, so adherence can count entries directly.

## The `training` skill

A standalone skill, a peer of `routines`. It is invoked directly (`/training`)
and by `daily-plan` / `weekly-plan` — mirroring how those skills already "run the
routines skill logic." It has three responsibilities:

1. **parse** — read recent messages from the training channel over a bounded
   window (e.g., the last 7 days, via
   `uv run alt-discord read <channel_id> --after <iso8601>`, or
   `alt_discord.reader.fetch_messages`), skip bot messages and messages already
   processed (their id present in some `workout_log`'s `source_message_ids`), and
   LLM-normalize each remaining message — appending its exercises to the matching
   workout for the same `logged_date` + `place` if one exists, otherwise creating
   a new `workout_log`. No stored cursor is needed; idempotency comes from
   `source_message_ids` dedupe, so re-reading an overlapping window is safe. An
   optional confirmation reply may be posted to the channel.
2. **today** — reconcile `config.training.plan` + today's weekday + today's
   calendar `FIT |` events + the irregularity algorithm to produce **today's
   workout** (its menu of exercises). Surface the last recorded numbers for that
   workout's key lifts so the user knows the targets to beat.
3. **review** — compute the current week's adherence against `weekly_base`
   (workouts grouped by place/type, with per-day hit/miss) and the progression of
   `key_lifts` / `kpi` over recent `workout_log` entries.

Keeping this logic in one skill isolates training concerns, keeps
`daily-plan` / `weekly-plan` thin, and follows the `routines` precedent.

### Irregularity algorithm

Encoded as prose in `config.training.plan`; applied by **today** and **review**:

- Sunday becomes basketball → slide Saturday to full rest.
- Basketball lands on Wednesday/Thursday → that day's gym/home training is
  paused; basketball counts as a max-intensity workout.
- Fatigue is high → skip optional home/floor work; the week is considered OK as
  long as the base (personal ×1 + 24h gym ×1–2) is met.

## Skill Integration

### `daily-plan`

- Phase 1 (gather): run `training` **parse** (so the day's view reflects last
  night's posts), then `training` **today**.
- Phase 2 (present): a new "Training" section — today's workout (menu);
  week-so-far vs base (e.g., Personal 1/1, 24h gym 1/2); last numbers for that
  workout's key lifts.
- Phase 4 (Discord summary): an optional 🏋 line for today's workout.

### `weekly-plan`

- A new "Training Review" section — adherence vs `weekly_base` (per-day
  hit/miss, base met?); progression of key lifts and KPI (e.g., deadlift
  60→62.5 kg; vertical jump latest vs previous); irregularity notes (basketball
  as max intensity); and next week's weekday menus reconciled with the calendar.

## Discord Logging UX

- A dedicated training channel; its id is stored in
  `config.training.discord.channel_id`. The user creates the channel and provides
  the id.
- One or more free-form posts per workout (e.g., one post per exercise is fine;
  the parser merges them by `logged_date` + `place`). The suggested format is
  loose and parsed flexibly by the LLM. Illustrative example:

  ```
  thu 24h gym lower
  bulgarian 20kg 10x3
  single-leg calf bodyweight 15x3
  vertical jump 60cm
  rpe8 left ankle tight
  ```

## Documentation Updates

- `.claude/rules/database-entries.md` — add the `workout_log` type to the
  overview table and the per-type reference section.
- `CLAUDE.md` — note `config.training` and the `/training` skill under
  configuration / key commands.

## Components Summary

| Component | New/Changed | Notes |
|---|---|---|
| `config.training` | new config key | markdown plan + machine hints |
| `workout_log` entry type | new convention | no migration (open `type` column) |
| `training` skill | new skill | parse / today / review; peer of `routines` |
| `daily-plan` skill | changed | add Training section + parse call |
| `weekly-plan` skill | changed | add Training Review section |
| `database-entries.md`, `CLAUDE.md` | changed | document the new type and config |

## Out of Scope (future, explicitly deferred)

- `workout-check-cloud` (push: reminders, missing-log detection, weekly
  auto-summary).
- Webapp progression charts.
- Automatic per-exercise 1RM estimation.
