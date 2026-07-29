# alt

alt is a personal planning application with a server-rendered Go web interface
and PostgreSQL as its source of truth.

The current application supports:

- daily and weekly plans with immutable revisions;
- guided daily planning with routines, public GitHub issues, optional Google
  Calendar context, and an OpenRouter model;
- routine categories, schedules, completion, and correction;
- Atlas-managed PostgreSQL schema migrations;
- liveness and readiness endpoints for container operation.

Authentication is not implemented yet. The local web service uses one
configured application user and must not be exposed publicly.

## Quick start

Start PostgreSQL, apply migrations, and run the web application:

```bash
docker compose up --build
```

Open <http://127.0.0.1:28080/>.

See [Getting Started](docs/getting-started.md) for the complete verified setup
and [Documentation](docs/index.md) for architecture, development procedures,
and design records.

## Testing

```bash
make test
```

## License

[MIT](LICENSE)
