# Getting Started

This guide is the shortest verified path from a fresh checkout to a working
local alt environment.

## Prerequisites

- Docker with Docker Compose
- Go 1.26 when running `make test` on the host
- `make`

Bitwarden Secrets Manager is optional for the basic local application. It is
required when testing configured AI or Google Calendar integrations.

## Start the application

From the repository root, run:

```bash
docker compose up --build
```

Compose:

1. starts PostgreSQL on `127.0.0.1:25432`;
2. starts Garage (S3-compatible object storage) on `127.0.0.1:23900`, which
   creates its dev bucket and access key on first start with no bootstrap step;
3. applies the versioned Atlas migrations;
4. builds and starts the Go web application on `127.0.0.1:28080`.

Open <http://127.0.0.1:28080/>.

The application uses a fixed local user by default. Do not expose this
unauthenticated development service publicly.

## Verify the application

Check process liveness:

```bash
curl --fail http://127.0.0.1:28080/livez
```

Check readiness, including PostgreSQL:

```bash
curl --fail http://127.0.0.1:28080/readyz
```

Run the Go test suite:

```bash
make test
```

## Configure integrations

The basic application starts without AI or Google Calendar credentials. To
inject development credentials from Bitwarden Secrets Manager, follow
[Bitwarden Secrets Manager Development Injection](development/bws-development.md).

Never put secret values in `.env` files, TOML configuration, Compose files, or
the repository.

## Stop the application

```bash
docker compose down
```

The PostgreSQL named volume remains available for the next start.

## Next steps

- Read the [Architecture Overview](architecture/overview.md).
- Follow the [Database Migrations](development/database-migrations.md)
  procedure before changing the schema.
