# Database Migrations

Atlas HCL files under `db/schema/` are the source of truth for the desired
PostgreSQL schema. Reviewed, versioned SQL files under `db/migrations/` are the
only migrations applied outside disposable schema-development databases.

## Format the schema

Use the Atlas version pinned in `compose.yaml`:

```bash
make atlas-fmt
```

## Generate a migration

Choose a concise snake-case migration name:

```bash
make atlas-diff MIGRATION_NAME=describe_change
```

The command compares the desired HCL schema with the existing migration
directory by using the disposable `schema-dev` PostgreSQL service.

## Review and verify

Before committing a generated migration:

1. review the SQL under `db/migrations/`;
2. confirm `db/migrations/atlas.sum` changed as expected;
3. run `make test`;
4. start a clean disposable environment when the migration needs end-to-end
   verification.

Do not apply an Atlas declarative diff directly to production. Production and
other persistent environments must apply only reviewed versioned migrations.
