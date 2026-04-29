# Config webapp UI (Phase 1: daily-plan)

Date: 2026-04-29
Status: Approved
Tracking: [#21](https://github.com/cloveclovedev/alt/issues/21)

## Goal

Make daily-plan configuration accessible to non-CLI users by surfacing it in
the webapp. As the proof of concept, expose the params consumed by `daily-plan`
and `daily-plan-cloud`, plus two new schedule-control params (`daily_plan.cloud.enabled`,
`daily_plan.cloud.fallback_time`). Establish the per-skill config UI mechanism
so later phases can extend to other skills (`weekly-plan`, `x-draft`, etc.).

## Philosophy

`config` is a key-value store of system settings, distinct in nature from
`entries` (an append-only log). The "single table" philosophy applies to
`entries`; for `config`, structured per-key metadata is appropriate.

The shipped repository carries a YAML catalog of the params each official skill
expects. The catalog is **not the runtime source of truth** — it is a
git-managed set of definitions used to bootstrap or update the database.
After bootstrap, the database is authoritative. Personal customisation
(adding a private key, editing a description, overriding a value) flows
through the same writes the seed CLI uses; it does not require touching the
YAML.

This preserves three properties at once:
- new clones can come up with sensible defaults (forker friendliness),
- official skill changes get reviewable diffs (catalog in git),
- individuals can extend the system freely without forking the repo.

## Non-goals (Phase 1)

- Editing param metadata (description / type / consumed_by) from the webapp.
  Phase 1 webapp edits values only.
- Adding a "create new param" UI in the webapp. Custom params are added via
  CLI (`alt-db config set` / `set-meta`) or via Claude Code conversations.
- Reconciling the cloud-scheduler cron schedule from config. Phase 1 reads
  `daily_plan.cloud.enabled` / `fallback_time` inside the existing dispatch
  loop only; the trigger cron itself stays as-is. Tracked in a follow-up
  issue.
- Editing the `routines` collection through atomic-key forms. The collection
  needs its own dedicated editor (separate issue).
- Other skills (`weekly-plan`, `x-draft`, `x-post`, `nutrition-check`,
  `wake-check`, `routines`). Phase 2.

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│  .claude/config-defaults.yaml                              │
│  (skill-author-edited initial param catalog, git-tracked)  │
└─────────────────┬──────────────────────────────────────────┘
                  │  uv run alt-db config seed (idempotent)
                  ▼
┌────────────────────────────────────────────────────────────┐
│  config table (Neon)                                       │
│    key | value (jsonb) | metadata (jsonb) | timestamps     │
│                                                            │
│  metadata = { type, description, consumed_by, default }    │
└──┬──────────────┬─────────────────────────┬────────────────┘
   │              │                         │
   ▼              ▼                         ▼
webapp           cloud-scheduler           daily-plan / other
(read+write      (read enabled +           skills
 value)           fallback_time)           (read as today)
```

Three write paths land in the same `config` table and are equal citizens:
- `alt-db config seed` (YAML → DB, insert-if-missing)
- `alt-db config set` / `set-meta` (CLI)
- webapp `/config` page (Server Action, value only in Phase 1)

The runtime truth is always the database row.

## Data model

### `config` table — `metadata` column

A new `metadata` jsonb column is added to the existing `config` table:

```sql
ALTER TABLE config
  ADD COLUMN metadata jsonb NOT NULL DEFAULT '{}'::jsonb;
```

Existing rows retain their `value` and receive `metadata = {}`.

`metadata` shape (all fields optional except `type`):

```json
{
  "type": "string | number | boolean | array | object",
  "description": "Human-readable explanation, may be multi-line.",
  "consumed_by": ["daily-plan", "weekly-plan"],
  "default": null
}
```

### YAML catalog — `.claude/config-defaults.yaml`

A single file at `.claude/config-defaults.yaml` (sibling to `skills/`) lists
the params shipped with the OSS repo. Skill-specific and shared params live
together; `consumed_by` carries the ownership information.

```yaml
params:
  plan.discord.channel_id:
    type: string
    description: |
      Discord channel ID where daily and weekly plans are posted.
    consumed_by: [daily-plan, weekly-plan]

  plan.github.repos:
    type: array
    description: |
      List of GitHub repositories (owner/name) to scan for issues during
      daily and weekly planning.
    consumed_by: [daily-plan, weekly-plan]

  plan.google_calendar.context:
    type: string
    description: |
      Free-form context describing how to interpret each calendar
      (e.g., "Work calendar = main job hours, mkuri = personal").
    consumed_by: [daily-plan, weekly-plan]

  daily_plan.cloud.enabled:
    type: boolean
    description: |
      When true, daily-plan-cloud runs as a fallback if you have not
      manually posted a daily plan by fallback_time.
    consumed_by: [daily-plan-cloud, cloud-scheduler]
    default: true

  daily_plan.cloud.fallback_time:
    type: string
    description: |
      JST time (HH:MM) at which daily-plan-cloud auto-posts if no daily plan
      has been posted yet. Cloud-scheduler dispatches based on the hour part.
    consumed_by: [daily-plan-cloud, cloud-scheduler]
    default: "10:23"
```

YAML never contains personal values (channel IDs, repo names). Defaults are
limited to safe, non-secret fallbacks.

## Components

| # | Component | Purpose |
|---|---|---|
| 1 | DB migration `<ts>_add_config_metadata.sql` | Add `metadata jsonb` column |
| 2 | `.claude/config-defaults.yaml` | Catalog of params shipped with the OSS skill set |
| 3 | `src/alt_db/config.py` | Add `set_meta(key, metadata)`, `list_with_meta()`, `load_yaml_defaults()`, `seed(force=False)` |
| 4 | `alt-db` CLI | New subcommands: `config seed [--force]`, `config set-meta <key> <json>`, `config list --with-meta` |
| 5 | `webapp/src/lib/config.ts` | Add `listConfigsWithMeta()`, `setConfigValues(updates)` |
| 6 | `webapp/src/app/config/` | New route: `page.tsx` (skill tabs index) and per-skill subroute / tab components |
| 7 | `webapp/src/app/config/actions.ts` | Server Action `saveConfigValues(updates: { key, value }[])`, auth-gated |
| 8 | `webapp/src/components/nav.tsx` | Add `Config` nav link |
| 9 | `.claude/skills/cloud-scheduler/SKILL.md` | Read `daily_plan.cloud.enabled` / `.fallback_time` before dispatching daily-plan-cloud |
| 10 | `README.md` | Add "Configuration" philosophy section explaining YAML catalog vs DB truth |
| 11 | Follow-up issues | Cron reconciler, webapp metadata editor, `routines` collection editor |

## Data flows

### A. New environment bootstrap (forker)

1. `atlas migrate apply` — config table receives `metadata` column.
2. `uv run alt-db config seed` — for each YAML param, insert a row if absent;
   set `metadata` from YAML; leave `value` at NULL when no `default`, otherwise
   at `default`. Existing rows are not touched.
3. Open webapp `/config` — params with no value display as "unset".
4. User fills values via the form — Server Action writes `value`.
5. `/daily-plan` runs as today; `getConfig(key)` reads value as before.

### B. Existing user (this project)

1. `atlas migrate apply` — column added; existing rows get `metadata = {}`.
2. `uv run alt-db config seed` — backfills `metadata` for keys present in YAML;
   does not touch `value`. Personal keys absent from YAML are untouched
   (`metadata` stays `{}`).
3. Webapp `/config` shows existing values; user can edit as needed.

### C. webapp value edit

```
form submit (global Save → all dirty fields)
  → Server Action saveConfigValues([{key, value}, ...])
  → auth check (existing GitHub OAuth allowlist)
  → for each update, cast value per metadata.type
  → run all UPDATEs in a single transaction
  → revalidatePath('/config')
  → return { ok, errors? } where errors maps key → message
```

### D. cloud-scheduler dispatch (new behaviour, daily-plan-cloud only in Phase 1)

The cloud-scheduler dispatch table stays hard-coded. Inside the existing
dispatch row for daily-plan-cloud, two reads are added:

```
enabled       = getConfig('daily_plan.cloud.enabled', default=true)
fallback_time = getConfig('daily_plan.cloud.fallback_time', default="10:23")
fallback_hour = parseInt(fallback_time.split(":")[0])

if enabled and current_jst_hour == fallback_hour:
    invoke daily-plan-cloud
else:
    skip
```

The cron schedule itself is not modified in Phase 1. If the user changes
`fallback_time` to an hour the unified trigger does not fire on, daily-plan-cloud
will not run until the cron is reconciled — that is the follow-up issue's job.

### E. Personal param addition (Phase 1: CLI / Claude Code only)

```
uv run alt-db config set my.custom.key '"value"'
uv run alt-db config set-meta my.custom.key \
  '{"type":"string","description":"my note","consumed_by":["myskill"]}'
```

The webapp displays the new param in the tab matching `consumed_by`. Params
whose `consumed_by` is empty or references an unknown skill appear in a
synthetic `Custom` tab so personal additions are not lost.

## Webapp UI

Top-level nav adds a `Config` entry. The `/config` page renders skill tabs:

- `daily-plan`
- `Custom` (params with empty / unknown `consumed_by`)

In Phase 1 only `daily-plan` and `Custom` exist; later phases add more
without changing the mechanism.

Inside each tab, params are listed in YAML / DB order. Phase 1 does not add
visual sub-grouping by dot-prefix. Each row shows:

- the full key (e.g., `plan.discord.channel_id`)
- the description from metadata (italic, smaller)
- a form input chosen by `metadata.type`:
  - `string` → `<Textarea>` if the current value contains a newline,
    otherwise `<Input>`
  - `number` → `<Input type="number">`
  - `boolean` → `<Switch>`
  - `array` → repeated `<Input>` rows with add/remove buttons (Phase 1
    assumes string items)
  - `object` → JSON `<Textarea>` (no structured editor in Phase 1)

A single global Save button at the bottom of each tab submits all dirty
fields in one Server Action call. Per-row Save is not added in Phase 1.

A param appearing in multiple `consumed_by` skills is shown identically in
each tab (it is the same key); editing in one tab updates the same DB row.

## Error handling

- **Type mismatch on save** — Server Action attempts cast per `metadata.type`,
  returns `{ ok: false, error }` on failure for the form to surface inline.
- **Missing metadata** — webapp infers a fallback type from the current
  `value`'s JSON type and shows `(no description set)` placeholder. Such params
  appear in the `Custom` tab.
- **YAML parse error during seed** — CLI exits non-zero, prints offending key,
  applies nothing (single transaction).
- **YAML key absent in DB** — `seed` inserts with NULL value (or `default`).
- **DB key absent in YAML** — `seed` does not touch it; preserves personal
  customisation.
- **Auth** — `/config` route protected by existing GitHub OAuth allowlist.
- **Concurrency** — last-write-wins via `updated_at`; acceptable for a
  single-user tool.

## Testing

- **Python (`tests/`)**:
  - `test_config_meta.py`: `set_meta`, `list_with_meta`, `seed (insert-if-missing)`,
    `seed --force` (overwrites metadata, preserves value).
  - Verify YAML parsing surfaces a clear error on malformed input.
- **Webapp (`webapp/src/__tests__/`)**:
  - `config.test.ts`: `listConfigsWithMeta()` returns expected shape; Server
    Action casts each `metadata.type` correctly.
- **Manual smoke (PR review)**:
  - `npm run dev` → open `/config` → edit each type of field → reload, confirm
    persistence.
  - Set `daily_plan.cloud.enabled = false`, invoke cloud-scheduler skill
    directly with the matching JST hour mocked; confirm daily-plan-cloud is
    skipped.

## Open follow-ups (separate issues)

- **Cron reconciler** — Claude Code skill (or `alt-db` subcommand) that reads
  every `*.cloud.fallback_time` from config and updates the unified
  cloud-scheduler trigger's cron via `CronCreate`/`CronList`/`CronDelete`.
  Required before adding cloud variants for skills whose hour differs from
  the current trigger schedule.
- **Webapp metadata editor** — UI to add new keys, edit description / type /
  consumed_by, alongside the value editing introduced in Phase 1.
- **`routines` collection editor** — dedicated UI for the `routines` JSON
  object (add / edit / remove individual routines), distinct from atomic-key
  forms.
- **Phase 2 skills** — extend the same mechanism to `weekly-plan`, `x-draft`,
  `x-post`, `nutrition-check`, `wake-check`.
