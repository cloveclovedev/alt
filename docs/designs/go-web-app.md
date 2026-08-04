# Go Web Application Foundation Design

- Status: Approved
- Branch: `refactor/go-web-app`

## Context

alt previously combined Claude Code skills, Python CLIs, Discord, a generic
`entries` table, and a separate Next.js application. That architecture was
useful for experimentation, but it made the product boundary implicit and made
typed validation, authorization, mobile API support, and deterministic
background work difficult to add.

The replacement starts with a dedicated Go web application and PostgreSQL. It
is intentionally a clean break: the previous runtime remains available in Git
history but does not coexist with the new foundation.

## Goals

1. Establish a small, reviewable Go and PostgreSQL foundation.
2. Make manual daily and weekly planning usable without AI or external
   integrations.
3. Keep the application portable as one stateless container.
4. Use provider-independent user IDs so authentication can be added without
   rewriting domain foreign keys.
5. Keep HTML and future JSON transports over the same application services.
6. Use Atlas HCL as the desired schema and deploy reviewed versioned SQL.

## Non-goals

- Journaling and task management before their behavior is designed.
- Authentication providers, public signup, teams, or billing.
- AI generation, calendar, GitHub, Discord, or notification integrations.
- A final visual design for the home page.
- Migrating legacy data into the new schema.

## Architecture

Use one Go module and one binary:

```text
backend/
  cmd/alt/main.go
  internal/
    core/
      config/
      database/
      httpserver/
      logging/
    identity/
      store.go
    planning/
      domain.go
      service.go
      store.go
      handler.go
      templates/
      static/
db/
  schema/
  migrations/
docs/
  designs/
```

The standard library `net/http` and `html/template` serve the initial HTML
interface. PostgreSQL access stays in feature stores, business validation stays
in services, and HTTP handlers remain transport adapters.

The web process exposes:

```text
/                                      current daily-plan home
/plans/{kind}/{period_start}            explicit planning period
/plans/{kind}/{period_start}/revisions  create a revision
/livez                                 process liveness
/readyz                                PostgreSQL readiness
```

The root page is the authenticated application home. The foundation renders the
current daily plan there; later features may compose the current weekly plan,
tasks, KPI progress, and other time-sensitive information into the same page.
The period routes remain the canonical detail pages.

## HTMX and loading-state convention

Any action that goes through the LLM or otherwise takes noticeable time must
show its waiting state in the UI rather than blocking on a full-page
navigation. This is a product-wide convention, not specific to one feature.

- Long or LLM-backed actions submit through HTMX (`hx-post`) instead of a
  plain form post, and the response replaces the same fragment the page
  rendered on initial load rather than issuing a redirect.
- While a request is in flight, the triggering button is disabled
  (`hx-disabled-elt`) and a small, action-specific label (for example
  "Sending…", "Confirming…") appears beside it via `hx-indicator`. The
  triggering control's position does not shift.
- Every HTMX-enhanced form keeps its plain `action`/`method` attributes, so
  the flow still works with JavaScript disabled: the server redirects back to
  the full page instead of returning a fragment when the request has no
  `HX-Request` header.
- htmx is vendored under a feature's `static/` directory and served from the
  application; the browser never depends on a CDN, matching the
  container-portability goal.

Reference: <https://htmx.org/docs/>

## Identity model

Identity, authentication mappings, and application profile data are separate
concepts.

### `users`

`users` is the provider-independent ownership and lifecycle anchor:

```text
id
status
created_at
updated_at
```

`status` uses the PostgreSQL `user_status` enum with the initial values
`active` and `disabled`.

### `profiles`

`profiles` owns application-facing personal data:

```text
user_id
display_name
timezone
created_at
updated_at
```

`profiles.user_id` is both the primary key and a foreign key to `users.id`,
which enforces at most one profile per user.

### Future authentication mapping

An authentication provider will add `user_identities` only when authentication
is implemented:

```text
user_id
issuer
subject
```

The pair `(issuer, subject)` will be unique. Domain tables will continue to
reference `users.id`, never a provider UID.

## Planning model

The model separates a planning period from its immutable revision history.

```text
plans
id
user_id
kind
period_start
created_at

plan_revisions
id
plan_id
revision
summary_markdown
content_markdown
created_at
```

`summary_markdown` is the concise form suitable for quick review or optional
publication. `content_markdown` is the complete plan.

`plan_kind` initially contains `daily` and `weekly`. Daily `period_start` is the
local plan date. Weekly `period_start` is Monday. The unique keys are
`(user_id, kind, period_start)` for a logical plan and `(plan_id, revision)` for
its revision history. The highest revision is current.

The application creates the logical plan when needed and locks that row while
allocating a revision, preventing concurrent writes from selecting the same
next revision.

Known future monthly and yearly planning can extend `plan_kind` while retaining
the same period and revision model. This model does not define the internal
structure of a plan: tasks, KPI targets, and other structured elements remain
separate future features that may reference `plans.id`.

## Schema decisions

- Use UUIDs for public ownership identifiers and `timestamptz` for instants.
- Use `date` for the start of a local planning period.
- Use a PostgreSQL enum only for a defined domain state, not as a placeholder
  for an unknown lifecycle.
- Keep summary and full plan text relational and explicit rather than embedding
  them in JSONB.
- Omit `journal_entries`, `tasks`, generic `entries`, and generic `config` from
  the clean baseline.
- Use composite unique indexes beginning with each foreign key, covering both
  ownership lookups and cascade operations.

## Schema workflow

The HCL files under `db/schema/` are the source of truth. The pinned Atlas
container generates a versioned baseline and future diffs:

```bash
make atlas-fmt
make atlas-diff MIGRATION_NAME=baseline
```

Generated SQL is reviewed and tested against an empty PostgreSQL database before
application. Production applies only versioned migrations; it never receives a
direct declarative schema apply.

References:

- <https://atlasgo.io/guides/evaluation/setup-migrations>
- <https://atlasgo.io/hcl/postgres>
- <https://www.postgresql.org/docs/current/datatype-enum.html>

## Deployment direction

The application has two environments: `dev` and `prod`. `prod`'s production stack
is defined in `deploy/prod/compose.yaml`, deployment-target agnostic: the web
container is reached only through a shared external reverse-proxy network
(`edge`) and publishes no host port. The first `prod` deployment runs on a home
Linux host, reachable only over Tailscale; the shared proxy, DNS, TLS, and the
private home-network topology are owned by a separate private infrastructure
repository. The same stack can later be exposed publicly (e.g. on a VPS) once
request authentication lands, or move to Cloud Run when managed operation
provides a concrete benefit.

PostgreSQL is never exposed publicly. Off-host backups and restore rehearsals
are required before the database contains data that matters.

## Deferred product areas

The following receive their own design before schema is added:

- task lifecycle and its relationship to plans;
- structured KPI targets and their relationship to monthly or yearly plans;
- journal or general daily-record semantics;
- AI proposal, provenance, model usage, and cost records;
- authentication and authorization;
- external context adapters;
- Flutter JSON API;
- hosted-service entitlements and billing.

## Decision log

This is a living design document. Each entry records a change to the design;
the sections above always describe the current intended design.

- 2026-07-27 — Initial foundation design: rebuild alt as a single Go web
  application with typed boundaries.
- 2026-08-04 — Added the `prod` environment and `deploy/prod/compose.yaml`
  (deployment-target agnostic: web reached only via the shared external `edge`
  network, no host port; PostgreSQL internal-only; Cloudflare R2 for object
  storage; secrets injected via `bws run`; a separate `prod` Google OAuth
  client). First deployed on a home Linux host behind a shared Caddy proxy over
  Tailscale, whose topology is owned by a separate private infra repo. `prod`
  stays tailnet-only until request authentication lands. Tracked in
  [#92](https://github.com/cloveclovedev/alt/issues/92); topology design in the
  private `home-infra` repo ([#71](https://github.com/cloveclovedev/alt/issues/71)).
- 2026-08-03 — Established the HTMX and loading-state convention: long or
  LLM-backed actions submit over HTMX and show an in-place, action-specific
  loading label next to the disabled triggering control instead of navigating
  on redirect; the plain-form fallback is preserved for no-JavaScript use.
  htmx is vendored rather than loaded from a CDN. Applied first to
  daily-planning's context refresh, continue-without-failed-sources, message
  send, and confirm actions, and to the planning home card's routine
  completion action. Tracked in
  [#65](https://github.com/cloveclovedev/alt/issues/65).
