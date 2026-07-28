# alt

alt is a personal planning application with a Go web interface and
PostgreSQL as its source of truth.

This branch is a clean foundation for rebuilding the product around a dedicated
web application. Historical Python, Next.js, Discord, and Claude skill
implementations remain available in Git history but are not part of the current
runtime.

## Current scope

- provider-independent `users`;
- application-owned `profiles`;
- daily and weekly `plans` with immutable `plan_revisions`;
- a server-rendered `/` home for the current daily plan;
- explicit period pages under `/plans/{kind}/{period_start}`;
- PostgreSQL liveness and readiness checks;
- Atlas declarative schema with reviewed versioned migrations.

Tasks, journals, authentication providers, AI generation, and external
integrations are intentionally deferred until their behavior is designed.

## Local development

Start PostgreSQL, apply migrations, and run the web application:

```bash
docker compose up --build
```

Open <http://127.0.0.1:8080/>.

The stack binds application and database ports to loopback. Authentication is
not implemented, so the local web service must not be exposed publicly.

Run Go tests:

```bash
make test
```

## HTTP endpoints

The current application exposes server-rendered HTML and operational endpoints.
There is no JSON API yet.

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/` | Render the configured user's daily plan for the current local date. |
| `GET` | `/plans/{kind}/{period_start}` | Render the latest revision for a daily or weekly planning period. |
| `POST` | `/plans/{kind}/{period_start}/revisions` | Save a new immutable revision for the planning period. |
| `GET` | `/static/*` | Serve embedded web assets. |
| `GET` | `/livez` | Report process liveness without checking dependencies. |
| `GET` | `/readyz` | Report readiness after pinging PostgreSQL. |

`kind` is currently `daily` or `weekly`. `period_start` uses `YYYY-MM-DD`;
weekly periods must start on Monday.

The revision endpoint accepts an `application/x-www-form-urlencoded` body:

| Field | Required | Maximum length | Purpose |
|-------|----------|----------------|---------|
| `summary_markdown` | Yes | 5,000 characters | Concise plan summary. |
| `content_markdown` | Yes | 50,000 characters | Complete plan. |

Authentication and `/api/v1` routes are deferred. Until authentication exists,
the configured local user owns every request.

## Schema workflow

The HCL files in `db/schema/` are the source of truth. Generate a reviewed
versioned migration with the pinned Atlas container:

```bash
make atlas-fmt
make atlas-diff MIGRATION_NAME=describe_change
```

Apply only migration files outside disposable development databases. Do not run
a declarative schema apply directly against production.

The architecture and current schema decisions are recorded in
[`docs/designs/2026-07-27-go-web-app-design.md`](docs/designs/2026-07-27-go-web-app-design.md).

## License

[MIT](LICENSE)
