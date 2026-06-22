---
name: training
description: Use to log workouts from the training Discord channel, see today's planned workout, or review weekly training adherence and progression
---

# Training

Tracks training: parses workout posts from a dedicated Discord channel into
`workout_log` entries, surfaces today's planned workout, and reviews weekly
adherence and key-lift progression. Invoked directly (`/training`) and by
`daily-plan` / `weekly-plan`.

## Terminology

- **training** — the program/domain.
- **workout** — one session; one `workout_log` entry, which may be assembled from
  several Discord posts.

## Configuration

Read the master plan and settings:
```bash
uv run alt-db config get training
```
Fields:
- `plan` — markdown master plan (canonical: places, weekday→menu, irregularity
  algorithm, safety notes).
- `key_lifts` — exercise identifiers tracked for progression.
- `kpi` — performance KPIs measured occasionally (e.g., vertical jump).
- `weekly_base` — `{ personal, gym_24h_min, gym_24h_target }` adherence rule.
- `discord.channel_id` — dedicated training channel. If empty, `parse` is a no-op
  (tell the caller the channel is not configured yet).

## Modes

Pick what the caller needs: `parse`, `today`, or `review`. The default for a bare
`/training` invocation is: run `parse`, then `today`.

### Mode: parse — ingest workout posts into `workout_log`

1. Current time in JST: `TZ=Asia/Tokyo date '+%Y-%m-%dT%H:%M:%S+09:00'`. Compute
   `<window_start>` = 7 days before now, ISO 8601.
2. If `discord.channel_id` is empty, stop and report "training channel not
   configured."
3. Read recent messages:
   ```bash
   uv run alt-discord read <channel_id> --after <window_start>
   ```
   (Equivalently `alt_discord.reader.fetch_messages('<channel_id>', after_timestamp='<window_start>')`.)
4. Drop messages where `author.bot == true`.
5. Build the processed-id set: list recent workout logs and collect every
   `metadata.source_message_ids`:
   ```bash
   uv run alt-db --json entry list --type workout_log --since 10d
   ```
   Skip any message whose id is already in that set.
6. For each remaining message, normalize with the LLM into:
   - `logged_date` — from the message timestamp in JST (unless the post states a
     date).
   - `place` — `personal | gym_24h | home_elevator | floor1 | basketball | other`.
   - `workout_type` — `upper | lower | full | hiit | basketball | other`.
   - `exercises[]` — `{ name, weight_kg?, reps?, sets?, value?, unit?, note? }`.
     Use `weight_kg`/`reps`/`sets` for resistance work; `value`/`unit` for
     measurements (e.g., a vertical-jump test `value:60, unit:"cm"`) and HIIT
     (rounds/duration).
   - `rpe?`, `note?`.
   Use the master `plan` to disambiguate `place`/`workout_type` and expected
   exercise names. Write all values in English.
7. Merge or create (preserves the one-entry-per-workout invariant):
   - If an existing `workout_log` already has the same `logged_date` AND `place`,
     append the new exercises to its `metadata.exercises` and add this message id
     to `metadata.source_message_ids`:
     ```bash
     uv run alt-db entry update <id> --metadata '<full merged metadata json>'
     ```
   - Otherwise create a new entry:
     ```bash
     uv run alt-db entry add --type workout_log \
       --title "Workout <logged_date> — <place label>" \
       --content "<raw message text>" \
       --metadata '{"logged_date":"<logged_date>","place":"<place>","workout_type":"<type>","exercises":[...],"source_message_ids":["<message_id>"],"estimated_by":"discord_parse"}'
     ```
8. (Optional) Post a one-line confirmation to the channel:
   ```bash
   uv run alt-discord post <channel_id> "🏋 logged: <short summary>"
   ```

### Mode: today — today's planned workout

1. Date and weekday in JST.
2. From `config.training.plan`, read this weekday's planned workout (place +
   menu).
3. Reconcile with today's calendar `FIT |` events and the irregularity algorithm
   in `plan`:
   - Sunday becomes basketball → Saturday slides to full rest.
   - Basketball on Wed/Thu → that day's gym/home work is paused; basketball is the
     workout.
   - High fatigue → optional home/floor work may be skipped; the week is OK if the
     base is met.
4. For each key lift in today's menu, fetch the last recorded numbers:
   ```bash
   uv run alt-db --json entry list --type workout_log --since 60d
   ```
   For each key lift, find the most recent `exercises[]` item whose `name` matches
   and show its `weight_kg`×`reps`×`sets` as "last time."
5. Compute this week's adherence so far (see `review` step 3) for the one-line
   "this week so far" counter.
6. Output:
   ```
   ### Today's Workout (<weekday>)
   <place> — <menu>
   - <exercise>: last <weight>kg <reps>x<sets>
   This week so far: Personal <n>/<base>, 24h gym <n>/<target>
   ```
   If today is a rest day (or a slide applies), say so explicitly.

### Mode: review — weekly adherence + progression

1. Compute `<monday>`..`<sunday>` for the current week (JST).
2. Fetch this week's workouts:
   ```bash
   uv run alt-db --json entry list --type workout_log --since 8d
   ```
   Keep entries whose `metadata.logged_date` falls in the week.
3. Adherence: count workouts grouped by `place` (personal, gym_24h, ...). Compare
   to `weekly_base` (`personal`, `gym_24h_min`/`gym_24h_target`). Report per-day
   hit/miss and whether the base was met. Count a basketball workout as
   max-intensity per the irregularity algorithm.
4. Progression: for each `key_lift` and `kpi`, list the recent trend across
   `workout_log` entries (last few data points; e.g., deadlift 60 → 62.5 kg;
   vertical jump latest vs previous). Pull from a wider window if needed:
   ```bash
   uv run alt-db --json entry list --type workout_log --since 60d
   ```
5. Output:
   ```
   ### Training Review (Week of <monday>)
   Adherence: Personal <n>/<base> · 24h gym <n>/<target> · base <met / not met>
   - Mon <hit/miss>: <summary> ...
   Progression:
   - deadlift: 60 → 62.5 kg
   - vertical jump: 60 cm (prev 58)
   ```

## Notes

- All `workout_log` content and metadata written in English.
- `workout_log` needs no schema migration — the `entries.type` column is an open
  string.
- Idempotency is by `metadata.source_message_ids`, so re-reading an overlapping
  window is safe.
