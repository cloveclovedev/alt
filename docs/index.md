# alt Documentation

This documentation covers local development, the current architecture, and the
design records that explain major decisions in alt.

## Start here

- [Getting Started](getting-started.md) — run and verify a local environment.
- [Architecture Overview](architecture/overview.md) — understand the current
  runtime, boundaries, and data flow.

## Development

- [Bitwarden Secrets Manager Development Injection](development/bws-development.md)
- [Database Migrations](development/database-migrations.md)

## Design records

Each design record is a living, per-feature document: it describes that
feature's current intended design, and a Decision log at the end records how the
design changed over time with links to the issues and pull requests that made
each change. Files are named for the feature, not dated. The
[Architecture Overview](architecture/overview.md) remains the source of truth
for the system as it exists now.

- [Go Web Application Foundation Design](designs/go-web-app.md)
- [Daily Planning Design](designs/daily-planning.md)
- [Daily Run Design](designs/daily-run.md)
- [Routine Management Design](designs/routine.md)
- [Nutrition Tracking Design](designs/nutrition-tracking.md)

## Work tracking

GitHub Issues are the source of truth for actionable and accepted deferred
work. Documentation restructuring is tracked in
[#45](https://github.com/cloveclovedev/alt/issues/45).
