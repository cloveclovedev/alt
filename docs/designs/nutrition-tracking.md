# Nutrition Tracking Design

- Status: Approved
- Depends on:
  - `go-web-app.md`
  - `daily-planning.md` (routine-status precedent, AI provider model)
- Design issue: [#69](https://github.com/cloveclovedev/alt/issues/69)

## Context

alt previously tracked daily nutrition through a Discord-driven skill
(`nutrition-check-cloud`) backed by `nutrition_items`, `nutrition_logs`, and
`nutrition_targets` in the Python/`alt_db` implementation. It parsed meal photos
and text with an LLM, kept a dictionary of frequent items, auto-calculated
targets from body composition (InBody), tracked supplements, and posted five
scheduled intraday check-ins to Discord. The Go rebuild (`go-web-app.md`) left
that runtime in Git history without migrating it, so no nutrition tracking
currently exists.

The user wants daily calorie and protein tracking plus AI coaching available
again, with low-friction logging, in the Go web application.

## Goals

1. Record daily calorie and protein intake in the Go web application.
2. Keep logging low-friction: photo + vision and free-text AI parsing from the
   start, alongside a manual form and a catalog quick-pick.
3. Show achievement against targets on the nutrition area, the home card, and in
   AI coaching.
4. Provide read-only AI coaching (achievement evaluation + adjustment
   suggestion) owned by the nutrition feature.
5. Keep the feature cleanly separable for OSS contribution: no dependency on
   other product features; shared concerns live in `core`, `platform`, or a
   foundational feature.
6. Let the user complete recording, review, and coaching from the home screen.

## Non-goals

- Micronutrients beyond calories and protein (the MVP tracks only these).
- Automatic target calculation from body composition (see Deferred work; #78).
- Intraday scheduled notifications (see Deferred work; #79).
- Supplement adherence tracking as a distinct behavior (#80).
- Long-term meal-photo retention and review (#81).
- Auto-registration proposals and natural-language catalog editing (#82).
- Daily-planning integration in this MVP; separated to #77.
- Weekly coaching (deferred until weekly planning exists).

## Interaction model

Nutrition is a recording UI plus a context source, not a scheduled nudger. It
has two surfaces:

- a dedicated area (`/nutrition`) for recording and management;
- a home-card fragment the composition surface at `/` assembles, so the primary
  loop (record, review totals vs target, view coaching) completes on the home
  screen. Detailed management (catalog, target history, corrections, past days)
  stays in `/nutrition`.

The home page is a composition surface (`go-web-app.md`): `internal/nutrition`
exposes fragments; the home page assembles them. Nutrition imports neither
`planning` nor the home package.

## Domain model

Three tables under a `nutrition_` prefix. Definitions (reusable catalog) and
occurrences (intake log) are named distinctly, mirroring the routine domain's
`routines` / `routine_events` split.

### `nutrition_catalog`

Pre-registered reusable items for quick entry and AI name matching.

```text
nutrition_catalog
id
user_id
name
calories_kcal
protein_g
kind            -- enum: food | supplement (default food)
source
created_at
updated_at
```

`(user_id, name)` is unique. `kind` classifies the item (supplements are
identifiable/filterable) without a dedicated adherence mechanism. Calories and
protein are per the registered serving.

### `nutrition_entries`

A mutable intake log, one row per consumed item. A "meal" is simply the rows
sharing `(logged_date, meal_type)`.

```text
nutrition_entries
id
user_id
logged_date     -- local date (profile timezone)
meal_type       -- enum: breakfast | lunch | dinner | snack
name
calories_kcal
protein_g
source          -- enum: manual | ai_photo | ai_text | catalog
catalog_id      -- nullable FK -> nutrition_catalog, ON DELETE SET NULL
created_at
```

`meal_type` is a property of the consumption event, not the catalog item: the
same catalog "protein" is `breakfast` when eaten as breakfast and `snack` when
taken in the evening. `name`, `calories_kcal`, and `protein_g` are denormalized
snapshots so editing or deleting a catalog item never rewrites historical
entries. Entries are correctable and deletable; unlike plan revisions they are
not immutable.

### `nutrition_targets`

Effective-dated daily targets. Only a start date is stored; the applicable
target is derived, following the routine domain's "derive, do not persist
redundant state" principle.

```text
nutrition_targets
id
user_id
effective_on    -- local date the target begins to apply
calories_kcal
protein_g
rationale       -- optional free text
created_at
updated_at
```

The target applicable to a date `D` is the row with the greatest
`effective_on <= D`. The current target is the applicable target for today.
Changing targets appends a new row; a newer `effective_on` implicitly supersedes
older ones, so there is no end column and no overlap/gap bookkeeping. A future
`effective_on` naturally does not take effect until its date. `effective_on` is
kept separate from `created_at` (audit) so backdated or future-dated targets keep
historical achievement rates correct.

All foreign keys are indexed; composite unique indexes begin with `user_id`.

## Logging input

Low-friction logging is a priority, so photo + vision is included from the start.
All AI input passes a confirmation gate before storage: the model only proposes
candidate entries; the user records them. This satisfies the "model has no
mutation capability" constraint (as with routine proposals, #60) — the user, not
the model, creates the log.

- Manual form: choose `meal_type`, enter name/kcal/protein (`source = manual`).
- Catalog quick-pick: create an entry from a catalog item's snapshot
  (`source = catalog`, `catalog_id` set).
- AI text: free text -> candidate entries with a guessed `meal_type`
  (`source = ai_text`).
- AI photo: upload -> temporary object storage -> vision extraction -> candidate
  entries (`source = ai_photo`).
- When a candidate name matches a catalog item, the catalog's stored values are
  preferred over the estimate.

Uploaded photos are stored temporarily in `core/objectstore` and discarded after
extraction; only structured entries are retained. Photos are served (while
present) via server-issued presigned URLs; keys never reach the browser. LLM
actions follow the HTMX loading-state convention (`go-web-app.md`).

## Coaching

Nutrition coaching knowledge belongs to the nutrition feature. A coaching
service in `internal/nutrition` (using the `internal/ai` foundation) produces,
from stored entries and the applicable target, an achievement-rate evaluation
(kcal/protein vs target) and an adjustment suggestion for the day. Coaching is
read-only commentary generated only from stored data; it never fabricates an
entry the user did not record.

Coaching is generated on demand and cached per `(user_id, date)` as a
regenerable derivative (not an immutable revision); a regenerate action re-runs
it. Scope is daily for the MVP. Output appears on the nutrition area and the
home-card fragment. The nutrition area's numeric achievement display (totals vs
target, percentages) is deterministic and needs no AI.

## AI foundation (shared)

Nutrition (vision parsing and coaching) and training (#70) both need inference,
so the shared AI concerns move out of `planning` into reusable locations
(tracked in #72; a prerequisite for #69 and #70):

- `platform/openrouter`: OpenRouter transport, auth, ZDR provider policy, and
  capability discovery (extended with image input modality); a generic
  completion call whose provider DTOs stop at the boundary.
- `internal/ai`: a foundational feature (a peer of `identity`) owning
  purpose-based model assignment (`ai_model_assignments`), usage metadata
  (`ai_generations`, no content), ZDR enforcement, and `/settings/ai` with
  per-purpose capability filtering. `planning` is refactored to consume it while
  preserving current behavior.

Two nutrition purposes are defined: `nutrition_logging` (requires a
vision-capable, structured-output, ZDR model; no fallback to a non-vision model)
and `nutrition_coaching` (text + ZDR). ZDR is enforced on every request, as in
daily planning.

Object storage uses `core/objectstore` (S3-compatible): Cloudflare R2 in
production, a local S3-compatible container (for example Garage) in dev/CI.
Swapping providers changes only config, so it is `core`, not `platform`.

## Package boundary

```text
internal/nutrition/   domain.go / service.go / store.go / handler.go
                      coaching.go (uses internal/ai) / templates/ / static/
internal/ai/          purpose model assignment, ai_generations, ZDR, /settings/ai
platform/openrouter/  transport + capability discovery (image modality)
core/objectstore/     S3-compatible object storage (R2 prod / Garage dev)
```

`internal/nutrition` depends only on the `internal/ai` and `core/objectstore`
foundations — never on `planning` or `routine`. If a cross-domain "health coach"
that reads nutrition, training, and routine data is wanted later, it becomes a
new feature consuming provider-neutral read ports, not a dependency added to
these features.

## Web surfaces

```text
/nutrition                 today's entries, totals vs target, coaching
/nutrition/entries         create (manual / catalog quick-pick / AI photo/text)
/nutrition/entries/{id}    edit / delete (corrections)
/nutrition/catalog         manage catalog items incl. kind
/nutrition/targets         set targets and view history
/nutrition/coaching        generate / refresh coaching (HTMX)
```

Plus a home-card fragment: today's totals vs target, a minimal quick-add
(AI text/photo or catalog quick-pick), and the day's cached coaching. Exact
handler paths may change during implementation; all transports call the same
service.

## Planning integration (separated to #77)

Daily-planning integration is delivered separately to keep the nutrition MVP
self-contained and the feature boundary clean. The intended design:

- `planning` declares a small port `NutritionContextSource` (analogous to
  `RoutineStatusSource`); `nutrition` implements it; `cmd/alt/main.go` wires it.
  `planning` never imports the `nutrition` package.
- The exposed facts are compact: yesterday's totals vs target (achievement
  rate), today-so-far totals, and a trailing 7-day average vs target — never a
  full entry dump.
- Nutrition is an independent gather source with the usual
  loaded/failed/unavailable status; planning still succeeds when nutrition is
  absent.
- The daily-planning prompt gains a compact nutrition-facts section so the plan
  can account for intake (for example favoring a high-protein dinner). The plan
  does **not** generate nutrition coaching commentary; coaching stays
  nutrition-owned. The prompt version advances when this contract changes.
- No `plan_revision_nutrition` component table is added: nutrition is context
  only, so the planning schema is unchanged. A nutrition to-do is an ordinary
  action item.

## Alternatives considered

- Free-text/manual only, deferring photo vision (rejected: photo capture is the
  main friction reduction the user valued; vision is included from the start,
  with a fallback to text/manual if it proves too heavy).
- Long-term photo retention (rejected for the MVP: adds object lifecycle and
  privacy surface; coaching works from structured data — deferred to #81).
- Coaching generated inside the daily-planning prompt (rejected: it would put
  nutrition domain knowledge into `planning` and couple the two; coaching is
  owned by `nutrition`, and planning receives only facts — #77).
- A separate `nutrition_coaching` feature (rejected now: it would depend on
  nutrition's data, the cross-feature coupling being avoided; coaching lives
  inside `nutrition`, depending only on the `internal/ai` foundation).
- Explicit target validity windows (`effective_from`/`effective_until` or an
  `active` boolean) (rejected: a single derived `effective_on` avoids
  overlap/gap bookkeeping and matches the routine "derive" principle).
- `supplement` as a `meal_type` (rejected: supplement-ness is an item property,
  not a time slot; it lives on `nutrition_catalog.kind`).

## Consequences

- Calorie and protein daily totals vs target are available for logging,
  home-card review, and coaching, with photo/text/manual/catalog entry paths.
- The AI layer becomes reusable (`platform/openrouter` + `internal/ai`),
  unblocking training coaching (#70) without duplication.
- Object storage is introduced (`core/objectstore`) for transient photo
  handling.
- Nutrition stays independent of `planning`/`routine`; planning integration and
  richer behaviors are additive follow-ups.

## Deferred work

- Body composition tracking, InBody import, and target auto-calculation — #78
  (near-term; targets are manual until this lands).
- Intraday check-in notifications via ntfy — #79.
- Supplement adherence tracking — #80.
- Meal photo retention for review — #81.
- Catalog auto-registration and natural-language management — #82.

## Follow-up implementation issues

- #72 — AI foundation refactor (`platform/openrouter` + `internal/ai`);
  prerequisite, shared with #70.
- #73 — nutrition schema and domain.
- #74 — logging input flow (form, catalog quick-pick, AI photo/text).
- #75 — on-demand daily coaching.
- #76 — web area and home-card fragment.
- #77 — expose nutrition context to daily planning.

## References

- OpenRouter structured outputs and ZDR (see `daily-planning.md` references).
- Cloudflare R2 (S3-compatible object storage).
- Garage (S3-compatible object storage for dev/CI):
  <https://garagehq.deuxfleurs.fr/>

## Decision log

This is a living design document. Each entry records a change to the design; the
sections above always describe the current intended design.

- 2026-08-03 — Initial nutrition-tracking design. Recording UI plus coaching in
  the Go web application; item-level `nutrition_entries` (calories and protein
  only) grouped by four `meal_type` time slots; a reusable `nutrition_catalog`
  with a `food`/`supplement` `kind`; effective-dated `nutrition_targets` deriving
  the applicable target from a single `effective_on`; low-friction logging via
  photo+vision, text AI, manual form, and catalog quick-pick behind a user
  confirmation gate, with photos discarded after extraction; nutrition-owned
  on-demand daily coaching cached per day; a shared AI foundation
  (`platform/openrouter` + `internal/ai`) and `core/objectstore`; a home-first
  interaction model; and planning integration separated as a port-based
  follow-up. Tracked in [#69](https://github.com/cloveclovedev/alt/issues/69);
  follow-ups #72–#82.
- 2026-08-03 — Implemented the shared AI foundation (#72), the prerequisite for
  the nutrition MVP. Added the `platform/openrouter` adapter (transport, ZDR
  provider policy, capability discovery extended with image-input modality, a
  generic `Complete`) and the `internal/ai` foundational feature (purpose-based
  model assignment, ZDR enforcement, per-purpose capability filtering on
  `/settings/ai`, and usage metadata). Registered the `daily_planning`,
  `nutrition_logging` (vision), and `nutrition_coaching` purposes; the logging
  purpose only offers image-capable ZDR models. Refactored `planning` to consume
  the foundation, removing its in-package OpenRouter client and AI tables. Made
  `ai_generations` feature-neutral by replacing its planning `session_id` foreign
  key with a `user_id` foreign key (backfilled from the referencing session), so
  any feature's generations share one usage table. Tracked in
  [#72](https://github.com/cloveclovedev/alt/issues/72).
- 2026-08-03 — Implemented the nutrition schema and domain (#73): the
  `nutrition_catalog`, `nutrition_entries`, and `nutrition_targets` tables and the
  `internal/nutrition` domain, store, and service (entries CRUD, catalog
  management, effective-dated targets, and daily and trailing-7-day summaries).
  Decisions made during implementation: `calories_kcal` is `integer` and
  `protein_g` is `numeric(6,1)` (read and summed as `float8` so pgx scans it into
  a plain `float64`); targets carry a unique `(user_id, effective_on)` constraint
  so re-setting a target for the same date replaces it via upsert rather than
  adding an ambiguous second row; the trailing average divides by the full window
  length, counting untracked days as zero. Tracked in
  [#73](https://github.com/cloveclovedev/alt/issues/73).
- 2026-08-03 — Implemented the logging input flow (#74): manual entry, catalog
  quick-pick, and AI photo/text parsing behind a user confirmation gate, in
  `internal/nutrition` (`logging.go`, `handler.go`, `templates/`). Added the
  `core/objectstore` S3-compatible adapter (aws-sdk-go-v2, path-style, endpoint
  override, presigned GET) and a local Garage service in Docker Compose
  configured entirely by env (`--single-node --default-bucket`), so no bootstrap
  step is needed. Decisions: the AI logging purpose (`nutrition_logging`) is used
  for both text and photo; for vision the image is sent inline as a base64 data
  URL rather than a presigned URL, because a dev object store is not reachable
  from the provider — the photo is still staged in object storage and deleted
  after extraction, satisfying the transient-retention decision. Candidates are
  never persisted server-side: they round-trip through an editable preview and
  are written only on confirmation. A malformed id path parameter now maps to
  not-found (400/404) rather than a 502. The engineering-policy default S3 test
  container was switched from MinIO to Garage (MinIO's server is unmaintained).
  Tracked in [#74](https://github.com/cloveclovedev/alt/issues/74).
