# Go Web Application and Daily Planning Design

- Status: Approved
- Date: 2026-07-27
- Branch: `refactor/go-web-app`

## 1. Context

alt currently treats Claude Code skills as the application layer. The skills
collect context from Google Calendar, GitHub, Discord, and PostgreSQL, then use
the generic `entries` table as a flexible record store. A Next.js application
reads that table directly.

This arrangement is useful for experimentation, but it makes the LLM prompt,
database schema, user interface, and external integrations one implicit
application boundary. It also makes typed validation, authorization, mobile API
support, and deterministic background work harder to add.

The new direction keeps the useful behavior and existing data while moving the
primary product experience to a dedicated Go web application. The product starts
as a private single-user tool but avoids database assumptions that would require
a second full rewrite for hosted multi-user use.

## 2. Goals

1. Make daily capture, planning, execution, and review usable from a dedicated
   mobile-friendly web interface.
2. Run HTML routes, future JSON API routes, and background work from one Go
   codebase and one container image.
3. Keep PostgreSQL as the source of truth for user records, confirmed memories,
   plans, and non-secret settings.
4. Preserve the existing Python tools, skills, `entries`, and `config` tables
   during an incremental migration.
5. Keep future Flutter clients thin by placing business behavior in Go services,
   not HTML handlers.
6. Keep deployment portable from a private home server to a VPS or managed
   container platform.

## 3. Non-goals

- Reimplement every existing skill in the first phase.
- Remove Discord, the Python CLIs, Next.js, or the legacy tables immediately.
- Add billing, teams, public signup, or a production identity provider now.
- Build vector search before ordinary relational queries prove insufficient.
- Copy PepperCheck's complete production infrastructure into the personal MVP.

## 4. Architecture Decisions

### 4.1 Modular monolith

Use one Go module and one binary with subcommands:

```text
alt web
alt worker
alt healthcheck
```

The `web` command serves:

```text
/                    server-rendered HTML
/today               daily planning interface
/api/v1/*             future Flutter-facing JSON API
/livez                process liveness
/readyz               dependency readiness
```

The web and worker commands use the same feature services and the same immutable
container image. Long-running generation and synchronization move to durable
jobs when introduced.

### 4.2 Package layout

```text
backend/
  cmd/alt/main.go
  internal/
    core/
      config/
      database/
      httpserver/
      logging/
    platform/
      ai/
      auth/
      calendar/
      github/
    dailyplan/
      domain.go
      service.go
      store.go
      handler.go
      templates/
      static/
    journal/
    task/
    routine/
    memory/
    settings/
db/
  schema/
  migrations/
```

Feature packages own their domain, application service, outbound PostgreSQL
store, and inbound HTTP handler. `core` contains shared provider-neutral
technical foundations. Shared provider-specific adapters belong in `platform`.
The composition root wires concrete implementations in
`backend/cmd/alt/main.go`. The module lives under `backend/` so Go tooling does
not interpret the existing Next.js route directories such as `[id]` as Go
import paths.

### 4.3 Server-rendered HTML

Use the standard library `net/http` router and `html/template`. Templates and
static assets are embedded in the binary. Add htmx only where partial HTML
responses materially improve an interaction; forms retain normal HTTP actions
so core flows work without JavaScript.

The standard library supports method-aware route patterns, and `html/template`
performs contextual escaping for untrusted data:

- <https://pkg.go.dev/net/http#ServeMux>
- <https://pkg.go.dev/html/template>

htmx expects HTML responses and documents progressive enhancement patterns:

- <https://htmx.org/docs/>

### 4.4 HTML and JSON share services

HTML handlers and future `/api/v1` handlers are separate inbound adapters. They
call the same application services and do not query PostgreSQL directly.
Flutter therefore adds a transport, not a second implementation of planning
rules.

### 4.5 External integrations are optional inputs

Google Calendar, GitHub, Discord, and AI providers are adapters around explicit
application contracts. A daily plan can still be created manually when any
integration is unavailable. Discord becomes an optional publication target,
not the primary user interface or capture system.

## 5. Data Model

### 5.1 Ownership

Create an internal `users` table now and put `user_id` on every user-owned
record. The private deployment bootstraps one configured user. A future
`user_identities` table maps verified external identities to the internal user
without changing domain foreign keys.

### 5.2 Initial typed tables

The first vertical slice adds:

- `users`: provider-independent ownership root.
- `journal_entries`: raw observations and notes with `occurred_at`.
- `tasks`: actionable work with typed status, priority, and due date.
- `daily_plans`: one versioned plan per user and local date.

Later phases add:

- `daily_plan_items`: ordered, typed items when the UI supports structured plans.
- `settings`: per-user non-secret preferences.
- `jobs`: durable background work and retry state.
- `ai_runs`: prompt/input lineage, model, token use, cost, output, and decision.
- `memories`: confirmed or proposed durable facts with provenance.
- `routines` and `routine_completions`.
- `user_identities`.

### 5.3 Typed columns and JSONB

Columns used for constraints, authorization, filtering, sorting, or lifecycle
transitions remain typed. JSONB is reserved for source-specific metadata and
immutable generation snapshots. PostgreSQL recommends JSONB for most JSON
storage, but that does not make it a replacement for stable relational fields:

- <https://www.postgresql.org/docs/current/datatype-json.html>

Use `timestamptz` for instants and `date` for a user's local planning date.
Index all foreign-key columns and the actual list-page access paths.

### 5.4 Settings and secrets

User preferences, prompt configuration, and confirmed memory belong in
PostgreSQL. API keys, database credentials, cookie keys, and provider tokens do
not. Secrets are injected at runtime through environment variables locally and
file-based secrets in deployed containers.

Committed defaults are seed input. PostgreSQL is authoritative after bootstrap.

### 5.5 Legacy compatibility

The `entries` and `config` tables remain intact during the new vertical slices.
No dual-write is added by default. A feature moves only after:

1. its new typed schema exists;
2. its Go service and UI are usable;
3. required legacy rows can be imported deterministically;
4. parity tests cover the retained behavior.

The existing declarative schema must first be reconciled with migrations:
`config.metadata` exists in the migration history but is missing from
`db/schema/config.hcl`.

## 6. Daily Planning Vertical Slice

### Phase A: Foundation

- Go module and single binary.
- Config, structured logging, PostgreSQL pool, HTTP lifecycle.
- `/livez`, `/readyz`, and an embedded responsive UI shell.
- Atlas declarative schema plus reviewed versioned SQL.

### Phase B: Manual daily loop

- `/today` shows today's journal entries, active tasks, and current plan.
- Add a journal entry from the page.
- Add a task from the page.
- Create or revise a daily plan explicitly.
- Keep planning useful without an AI or external account.

### Phase C: Assisted planning

- Add durable generation jobs and an `ai_runs` audit record.
- Generate a proposal from an immutable input snapshot.
- Require review before replacing the accepted plan.
- Record estimated provider cost and usage from the first hosted model call.

### Phase D: Context adapters

- Read calendar events through a calendar boundary.
- Read repository work through a GitHub boundary.
- Import selected legacy tasks and routines.
- Add optional Discord publishing after the web workflow is complete.

### Phase E: Mobile API

- Add authenticated JSON DTOs under `/api/v1`.
- Reuse the same services and authorization checks.
- Build Flutter only after the mobile web experience identifies native-only
  requirements.

## 7. Schema Workflow

Keep HCL split by feature as the desired schema. Generate versioned migrations
with `atlas migrate diff`, review the SQL, test it against an empty PostgreSQL
database, and apply only the versioned SQL outside disposable local development.

Atlas documents this combined declarative and versioned workflow:

- <https://atlasgo.io/concepts/declarative-vs-versioned>
- <https://atlasgo.io/versioned/diff>

The project-pinned containerized Atlas CLI is preferred for generation,
formatting, linting, and application.

## 8. Deployment Ladder

### Private use

Run Docker Compose on a home Linux host:

```text
web + worker + postgres + backup
```

Expose the web process only through Tailscale. Bind PostgreSQL only to the
internal Compose network or loopback. Back up off-host and rehearse restores.

### Public beta

Use a small VPS with:

```text
caddy + web + worker + postgres + backup
```

Build images in CI, deploy immutable digests, keep port 5432 private, inject
secrets at runtime, and add liveness, readiness, backup-age, and disk alerts.

### Growth

Move PostgreSQL to a managed provider when backup, recovery, availability, or
operator workload justifies the fee. Move the stateless image to Cloud Run or
another container platform only when it reduces real operational burden.

## 9. OSS and Commercial Boundaries

Develop the private product first, but keep it "single user deployed,
multi-user capable":

- internal user ownership from the first typed table;
- provider-neutral auth boundary;
- export and deletion paths;
- AI usage and cost records;
- an entitlement boundary that can remain inactive initially.

The likely business model is:

- free self-hosted OSS core;
- paid hosted service for managed operation, backups, synchronization,
  notifications, integrations, and included AI usage;
- optional bring-your-own-key support for self-hosting.

Billing is deliberately deferred until the daily loop has repeated personal
value. Before accepting outside contributions, review whether the current MIT
license matches the desired hosted-service strategy. Data export must not be a
retention paywall.

## 10. First Implementation Boundary

The first implementation on this branch will:

1. reconcile `config.metadata` in the declarative schema;
2. add the Go foundation and health endpoints;
3. add the initial typed daily-planning schema;
4. render `/today` from PostgreSQL;
5. support journal-entry capture and task creation/completion through ordinary
   HTML forms;
6. add unit and PostgreSQL integration coverage where practical;
7. leave legacy code and tables operational.
