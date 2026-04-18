# Second Brain Knowledge Store — Design

## Problem

alt has planning and routine management, but no structured way to persist and resurface knowledge — casual thoughts, goals, tech interests, business objectives, and lessons learned. Information written in files gets forgotten because nothing proactively surfaces it.

## Solution

Add a cloud-hosted knowledge store (Neon Postgres) with a Python CLI for access, integrate it into existing skills for automatic surfacing, and build a Next.js dashboard for visual overview.

## Architecture: Hybrid (DB + Files)

| Data | Storage | Reason |
|---|---|---|
| Routine definitions | YAML (keep as-is) | Human-editable, git-managed |
| Routine events | Neon Postgres (migrate from SQLite) | Append-only event log, time-series queries |
| Memos / casual thoughts | Neon Postgres | Searchable, auto-surfacing |
| Goals / objectives | Neon Postgres | Status tracking, deadline reminders |
| Tech interests | Neon Postgres | Tag search, change history |
| Knowledge / lessons learned | Neon Postgres | Full-text search, cross-referencing |
| Project config | TOML (keep as-is) | Low change frequency, git-managed |

**Separation principle:** Mutable knowledge store (`entries`) vs. append-only event log (`routine_events`). Files remain for configuration and definitions.

## Database

### Host

Neon (serverless Postgres). Free tier: 0.5GB storage, 190 compute hours/month.

### Schema Management: Atlas (Declarative)

Desired state defined in HCL files. Atlas computes and applies diffs — no manual migration files.

```
db/
  atlas.hcl           # Atlas config (connection, env)
  schema/
    entries.hcl        # entries table + indexes
    routine_events.hcl # routine_events table + indexes
```

Apply workflow:

```bash
atlas schema apply --env neon
```

`atlas.hcl` reads `DATABASE_URL` from environment. Schema files are git-managed.

### Schema

**entries table** (`db/schema/entries.hcl`):

```hcl
table "entries" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "type" {
    type = text
    null = false
    comment = "memo, goal, knowledge, tech_interest, business"
  }
  column "title" {
    type = text
    null = false
  }
  column "content" {
    type = text
    null = true
    comment = "Body text, Markdown supported"
  }
  column "status" {
    type = text
    null = true
    comment = "For goals: active, achieved, dropped"
  }
  column "tags" {
    type    = jsonb
    default = "[]"
  }
  column "metadata" {
    type    = jsonb
    default = "{}"
    comment = "Type-specific fields (target_date, revenue_target, etc.)"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_entries_type" {
    columns = [column.type]
  }
  index "idx_entries_tags" {
    type    = GIN
    columns = [column.tags]
  }
  index "idx_entries_created" {
    columns = [column.created_at]
  }
}
```

**routine_events table** (`db/schema/routine_events.hcl`):

```hcl
table "routine_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "routine_name" {
    type = text
    null = false
  }
  column "category" {
    type = text
    null = false
    comment = "household, dog, health"
  }
  column "completed_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "kind" {
    type    = text
    null    = false
    default = "completed"
    comment = "completed | baseline"
  }
  column "note" {
    type = text
    null = true
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_routine_events_name" {
    columns = [column.routine_name]
  }
}
```

**Design decisions:**
- Single `entries` table for all knowledge types. Data volume is personal-scale; simplicity over normalization.
- `type` and `status` are TEXT, not enum. Postgres enums require `ALTER TYPE` to add values; TEXT allows organic evolution. Validation lives in the application layer.
- `metadata` JSONB for type-specific fields (e.g., goal: `{"target_date": "2026-09", "revenue_target": 300000}`).
- `routine_events` (not `routine_completions`) because it holds both `completed` and `baseline` kinds. Baseline sets a tracking start point without faking a completion.
- No relationship table for now. Add in a future phase if link-traversal needs arise.

## Data Access: Python CLI (`scripts/db.py`)

Replaces `scripts/db.sh` (SQLite). Single entry point for all DB operations.

**Dependencies:** `psycopg2-binary`, `python-dotenv`. Managed with `uv`.

**Connection:** Reads `DATABASE_URL` from `.env` via `python-dotenv`. Never outputs the connection string.

```
# entries
python scripts/db.py entry add --type memo --title "..." --content "..." --tags '["peppercheck"]'
python scripts/db.py entry list --type goal --status active
python scripts/db.py entry list --since 7d
python scripts/db.py entry search "Flutter"
python scripts/db.py entry update <id> --status achieved
python scripts/db.py entry delete <id>

# routine_events
python scripts/db.py routine complete "Clean the toilet" household
python scripts/db.py routine baseline "Take flea medicine" dog --date 2026-05-01
python scripts/db.py routine last "Clean the toilet"
python scripts/db.py routine all
```

Output format: plain text for human/LLM readability. Optional `--json` flag for programmatic use.

## Skill Integration

### daily-plan additions

Information gathering phase adds:
- Active goals: `python scripts/db.py entry list --type goal --status active`
- Recent memos (last 7 days): `python scripts/db.py entry list --since 7d`
- Goals with approaching deadlines: `python scripts/db.py entry list --type goal --due-within 7d`

Display: concise summary, only today-relevant items. Avoid noise.

### weekly-plan additions

- All active goals with status overview
- Memos and knowledge entries added last week
- Alert for goals with no status change in 30+ days

Display: broader overview for weekly reflection.

## Web Dashboard

**Stack:** Next.js (App Router) on Vercel. Neon official integration for DB connection.

**Pages (minimum viable):**

| Route | Content |
|---|---|
| `/` | Dashboard — active goals, recent memos, deadline alerts |
| `/entries` | Entry list — filter by type/tag/status, full-text search |
| `/routines` | Routine status — last completion dates, overdue indicators |

**Initial scope:** Read-only. Write operations added later.

**Authentication:** Vercel Authentication (single user, personal project).

## Security

| Layer | Measure |
|---|---|
| Neon connection | SSL required (Neon default). Connection string in `.env` only, gitignored |
| `.env` protection | `sensitive-files.md` rule blocks Claude Code Read tool from `.env` files |
| LLM boundary | LLM never sees `DATABASE_URL`. Python process reads it internally, only query results go to stdout |
| Web dashboard | Vercel Authentication — only authenticated user can access |
| DB permissions | Application DB role: INSERT, SELECT, UPDATE, DELETE only. No DDL (CREATE, DROP, ALTER) in production |
| Data encryption | Neon provides encryption at rest (AES-256) and in transit (TLS) |
| Claude Cloud | `DATABASE_URL` set as project environment variable. Same LLM boundary applies — scripts only |
| Credential rotation | Neon supports connection string regeneration. Rotate if suspected compromise |
| Atlas | Schema changes applied via `atlas schema apply` from local machine only. Production DB role for Atlas is separate from application role |

## Migration Path

1. Set up Neon project and create Atlas config
2. Define schema in HCL and apply with `atlas schema apply`
3. Implement `scripts/db.py` with `uv`
4. Migrate existing `routine_completions` data from SQLite to `routine_events` in Neon
5. Update `.claude/skills/routines/SKILL.md` to use new CLI
6. Add entry queries to daily-plan and weekly-plan skills
7. Build and deploy Next.js dashboard
8. Deprecate `scripts/db.sh`

## Out of Scope (Future)

- Relationship table between entries (when link-traversal needs arise)
- pgvector semantic search (when "which memo was that?" becomes frequent)
- Entry creation from web dashboard (after read-only is validated)
- YAML/TOML format unification (separate task)
