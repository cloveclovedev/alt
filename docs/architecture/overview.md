# Architecture Overview

alt is a single Go application backed by PostgreSQL. It serves a
server-rendered web interface, performs application orchestration, and connects
to optional external planning sources.

```text
Browser
  |
  v
Go HTTP server
  |-- planning feature
  |-- routine feature
  |-- identity bootstrap
  |-- embedded templates and static assets
  |
  +--> PostgreSQL
  +--> GitHub public REST API
  +--> Google Calendar API
  +--> OpenRouter API
```

## Runtime

`backend/cmd/alt/main.go` is the composition root. The binary supports:

- `alt web` — connect to PostgreSQL, wire the application, and serve HTTP;
- `alt healthcheck` — check the local process liveness endpoint.

The HTTP server shuts down gracefully on `SIGINT` or `SIGTERM`. Structured logs
are written to stdout. HTML templates and CSS are embedded in the binary.

Docker Compose starts PostgreSQL, applies versioned migrations, and runs the
same stateless application image used by other container environments.

## Packages and boundaries

```text
backend/
├── cmd/alt/                 composition root
└── internal/
    ├── core/                provider-neutral technical foundations
    │   ├── config/
    │   ├── database/
    │   ├── httpserver/
    │   ├── logging/
    │   └── objectstore/     S3-compatible object storage (R2 prod / Garage dev)
    ├── platform/            shared, provider-specific integrations
    │   └── openrouter/      OpenRouter transport, ZDR policy, capability discovery
    ├── ai/                  shared inference foundation: purpose-based model
    │                        assignment, ZDR enforcement, usage metadata, /settings/ai
    ├── identity/            application users and local bootstrap
    ├── nutrition/           calorie/protein logging, catalog, targets, AI parsing
    ├── planning/            plans, daily sessions, and source adapters
    └── routine/             routine definitions and completion events
```

Feature services own business rules and orchestration. HTTP handlers translate
requests and responses. Stores own PostgreSQL access. External provider
responses are normalized into application-owned models before they reach feature
rules.

`ai` is a foundational feature (a peer of `identity`) that any feature depends on
for inference without importing another feature. It sits in front of the
`platform/openrouter` adapter, which stops OpenRouter response DTOs at the
boundary. A provider integration used by only one feature stays inside that
feature; it moves to `internal/platform/` once it becomes shared, as OpenRouter
did when nutrition and training joined daily planning as inference callers.

## Identity and authorization

Domain tables reference provider-independent application user IDs. The local
runtime bootstraps one configured user and injects that user ID into feature
services.

Firebase Authentication and request-scoped users are not implemented. Until
they are, the web service is for loopback-only development and must not be
exposed publicly. Authentication work is tracked in
[#46](https://github.com/cloveclovedev/alt/issues/46).

## Planning

Plans identify a user, a period kind, and a period start. Their content is
stored as immutable numbered revisions.

Daily planning adds a resumable session state machine:

```text
gathering
  |
  +--> context blocked -- explicit continuation --> chatting
  |
  +-----------------------------------------------> chatting
                                                     |
                                                     v
                                                  reviewing
                                                     |
                                                     v
                                                  finalized
```

The session gathers normalized context from enabled Calendar sources, public
GitHub repositories, and routines. User and assistant messages are persisted
while the session is active. A structured preview is validated against saved
context before confirmation creates an immutable plan revision.

## Routines

The routine feature manages categories, recurring routine definitions,
completion events, corrections, and due-date calculation. Daily planning reads
normalized routine status through the routine service rather than querying its
tables directly.

## Web surfaces

The application exposes server-rendered routes in these groups:

- `/` and `/plans/{kind}/{period_start}` for current and historical plans;
- `/planning/daily/{date}` for the daily planning session;
- `/routines` and `/routine-categories` for routine management;
- `/settings/calendar`, `/settings/github`, and `/settings/ai` for planning
  integrations;
- `/livez` and `/readyz` for container health.

There is no public JSON API yet.

## Data and schema

PostgreSQL is the source of truth. Feature stores contain SQL and transaction
behavior; services and handlers do not.

Desired schema is declared under `db/schema/`. Atlas generates reviewed
versioned SQL under `db/migrations/`. See
[Database Migrations](../development/database-migrations.md).

## Configuration and secrets

Configuration resolves in this order:

```text
defaults -> TOML -> environment
```

Non-secret local defaults are documented in
`backend/config/alt.example.toml`. Deployment-selected values such as the
listen port and database URL may use environment overrides.

Secrets support environment variables and `*_FILE` injection. Development
integration secrets come from Bitwarden Secrets Manager; persistent secret
values do not belong in the repository or TOML configuration.

## Design history

Current behavior is summarized here. Context and rationale for major decisions
remain in the permanent [design records](../index.md#design-records).
