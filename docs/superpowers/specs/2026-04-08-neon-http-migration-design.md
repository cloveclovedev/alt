# Neon HTTP Migration Design

Migrate `alt-db` CLI from psycopg2 (TCP) to Neon's SQL-over-HTTP API (HTTPS) to resolve connectivity failures in Claude Code Cloud scheduled tasks.

## Problem

`psycopg2-binary` bundles `libpq`, whose DNS resolver fails in Cloud container environments. Discord (HTTPS) works fine in the same environment, confirming HTTP connectivity is available.

## Solution

Replace `psycopg2` TCP connections with HTTP POST requests to Neon's `/sql` endpoint using Python's `urllib.request` (stdlib). No new runtime dependencies.

```
Before:  alt-db CLI -> psycopg2 -> TCP:5432 -> Neon PostgreSQL
After:   alt-db CLI -> urllib.request -> HTTPS:443 -> Neon HTTP API (/sql)
```

## Environment Variables

Replace single `DATABASE_URL` with individual variables:

| Variable | Description |
|---|---|
| `NEON_HOST` | Neon hostname (e.g. `ep-xxx.us-east-2.aws.neon.tech`) |
| `NEON_DATABASE` | Database name |
| `NEON_USER` | Database user |
| `NEON_PASSWORD` | Database password |

These must be set in both local `.env` and the Cloud environment configuration.

## File Changes

### `src/alt_db/connection.py` (rewrite)

```python
class NeonHTTP:
    """Neon SQL-over-HTTP client."""

    def __init__(self, host: str, database: str, user: str, password: str):
        self.host = host
        self.database = database
        self.user = user
        self.password = password

    @classmethod
    def from_env(cls) -> "NeonHTTP":
        """Create client from NEON_* environment variables."""
        load_dotenv()
        # Read NEON_HOST, NEON_DATABASE, NEON_USER, NEON_PASSWORD
        # Raise RuntimeError if any are missing

    def execute(self, sql: str, params: list | None = None) -> list[tuple]:
        """Execute SQL via HTTP POST to /sql endpoint.

        Returns list of row tuples. For INSERT/UPDATE/DELETE without
        RETURNING, returns an empty list.
        """
        # POST https://{host}/sql
        # Body: {"query": sql, "params": params or []}
        # Auth: verify exact method against Neon HTTP API docs during implementation
        # Parse response JSON, return rows as list[tuple]
```

### `src/alt_db/entries.py` (modify)

- Function signatures: `conn` -> `db: NeonHTTP`
- SQL parameters: `%s` -> `$1, $2, ...`
- Remove cursor pattern, use `db.execute()` directly
- Return types unchanged

Example:
```python
# Before
def add_entry(conn, type, title, ...):
    cur = conn.cursor()
    cur.execute("INSERT INTO entries (...) VALUES (%s, %s, ...) RETURNING id", (type, title, ...))
    entry_id = str(cur.fetchone()[0])
    cur.close()
    return entry_id

# After
def add_entry(db, type, title, ...):
    rows = db.execute("INSERT INTO entries (...) VALUES ($1, $2, ...) RETURNING id", [type, title, ...])
    return str(rows[0][0])
```

### `src/alt_db/routines.py` (modify)

Same pattern as entries.py: `conn` -> `db`, `%s` -> `$N`, cursor -> `db.execute()`.

### `src/alt_db/cli.py` (modify)

- `get_connection()` -> `NeonHTTP.from_env()`
- Remove `conn.commit()` and `conn.rollback()` (HTTP auto-commits per request)
- Error handling simplified: failed HTTP requests don't commit

### `pyproject.toml` (modify)

- Remove `psycopg2-binary` from `dependencies`
- Keep `python-dotenv`
- No new runtime dependencies

## Testing Strategy

### Problem

Current tests use psycopg2's transaction rollback for cleanup:
```python
connection.autocommit = False
yield connection
connection.rollback()
```

HTTP requests are stateless — each is an independent auto-committed transaction. Rollback across requests is not possible.

### Solution: DELETE-based cleanup

```python
@pytest.fixture
def db():
    client = NeonHTTP.from_env()
    created_entry_ids = []
    created_routine_names = set()
    yield client, created_entry_ids, created_routine_names
    # Teardown: delete test data
    for eid in created_entry_ids:
        client.execute("DELETE FROM entries WHERE id = $1", [eid])
    for name in created_routine_names:
        client.execute("DELETE FROM routine_events WHERE routine_name = $1", [name])
```

- `test_entries.py` / `test_routines.py`: track created IDs/names, DELETE in teardown
- `test_cli.py`: no DB access, unchanged
- Tests exercise the real HTTP transport — more valuable than mocking

### Test file changes

| File | Change |
|---|---|
| `tests/conftest.py` | Replace psycopg2 fixture with NeonHTTP + cleanup |
| `tests/test_entries.py` | Update fixture usage, track created IDs |
| `tests/test_routines.py` | Update fixture usage, track created names |
| `tests/test_cli.py` | No changes (no DB dependency) |

## Out of Scope

- Webapp changes (`webapp/src/lib/db.ts` already uses `@neondatabase/serverless`)
- Schema migrations
- Batch/transaction support via HTTP (not needed for current CRUD operations)
