# Universal Entries Table Design

## Problem

alt currently uses 7 database tables across different domains (entries, routine_events, body_measurements, body_measurement_goals, nutrition_items, nutrition_logs, nutrition_targets). When someone forks alt and wants to customize skills for their own lifestyle, they can't just tweak a skill — they also need to modify the DB schema, write migrations, and update CLI code. This makes the barrier to customization unnecessarily high.

## Key Insight: LLM as the Interpretation Layer

Traditional applications need strict schemas because **code interprets data** — types, constraints, and foreign keys prevent bad data from causing runtime errors.

In alt, **the LLM interprets data**. Claude Code reads metadata flexibly and adapts to whatever structure it finds. The effective "schema definition" lives in skill instructions (natural language), not in database columns. This means:

- Schema rigidity provides safety in traditional apps, but **restricts flexibility** in LLM-powered apps
- Adding a new data type means writing a new skill, not running a migration
- The DB's role shifts from "structured data store" to "secure persistent storage"

This approach is sound under alt's specific conditions:

- **Single-user system** — schema inconsistency only affects the owner
- **Low write volume** — a few to a few dozen entries per day
- **Frequent schema evolution** — data structures change as lifestyle changes
- **LLM query generation** — Claude Code generates the SQL, so JSONB access patterns like `(metadata->>'weight_kg')::numeric` are not a burden

It would NOT be appropriate for multi-tenant SaaS, high-throughput systems, or applications where non-LLM code directly consumes database output.

## Design

### Schema

```sql
CREATE TABLE entries (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type       TEXT NOT NULL,
    title      TEXT NOT NULL,
    content    TEXT,
    status     TEXT,
    metadata   JSONB NOT NULL DEFAULT '{}',
    parent_id  UUID REFERENCES entries(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_entries_type ON entries (type);
CREATE INDEX idx_entries_created ON entries (created_at);
CREATE INDEX idx_entries_parent ON entries (parent_id) WHERE parent_id IS NOT NULL;
```

8 columns total. Down from 7 tables with 50+ columns.

### Column Selection Criteria

A column is a fixed column if and only if it is universally meaningful across all entry types:

| Column | Rationale |
|--------|-----------|
| `type` | Entry kind. The primary query discriminator |
| `title` | Every entry has a name/title |
| `content` | Body text (nullable) |
| `status` | Lifecycle state — active/draft/completed/posted etc. Values vary by type |
| `metadata` | All type-specific data as flexible JSON |
| `parent_id` | Structural relationship between entries. Required for JOIN and recursive CTE performance — cannot live in metadata |
| `created_at` | Audit and time-series ordering |
| `updated_at` | Change tracking |

Things intentionally excluded from fixed columns:

- **tags** — With `type` and `status` providing primary filtering, tags are a niche use case (article-like content). Can live in metadata with a GIN index if needed later.
- **ref_id / ref_date** — Creates decision fatigue ("should this go in ref_id or metadata?"). All type-specific fields belong in metadata, no exceptions.

### Parent ID and Tree Structure

`parent_id` is the one structural relationship that warrants a fixed column:

- Recursive CTEs for tree traversal require a real column for practical performance
- Foreign key constraint (with `ON DELETE SET NULL`) provides referential integrity
- Tree structures are universally useful across types (threads, sub-tasks, grouped entries)

`ON DELETE SET NULL` is chosen over `CASCADE` because deletion behavior is use-case dependent. Orphaned children are safer than accidental cascading data loss. Application-level logic can implement stricter behavior per type.

### Type-Specific Constraints

Type-specific constraints (e.g., unique deduplication indexes) are not included in the initial schema. They can be added as partial expression indexes when needed:

```sql
-- Example: prevent duplicate Discord message processing
CREATE UNIQUE INDEX idx_nutrition_log_dedup
  ON entries ((metadata->>'source_message_id'))
  WHERE type = 'nutrition_log' AND metadata->>'source_message_id' IS NOT NULL;
```

This approach allows forkers to add only the constraints they need, without inheriting constraints for features they don't use.

### JSONB Performance at Personal Scale

Numeric aggregation via JSONB (e.g., `SUM((metadata->>'calories_kcal')::numeric)`) is a common concern. At personal data volumes, it is a non-issue:

- nutrition_logs: ~1,500 rows/year, daily aggregation hits ~5 rows
- body_measurements: ~100 rows/year
- 10-year accumulation: ~16,000 total rows

PostgreSQL JSONB is binary format — no text parsing overhead. Sub-millisecond for typical queries. If performance becomes a concern, expression indexes resolve it without schema changes:

```sql
CREATE INDEX ON entries (((metadata->>'calories_kcal')::numeric)) WHERE type = 'nutrition_log';
```

### Type Conventions

Each entry type defines its own metadata structure via skill instructions. The database does not enforce metadata shape. Examples:

**nutrition_log:**
```json
{
  "logged_date": "2026-04-22",
  "meal_type": "lunch",
  "calories_kcal": 650.0,
  "protein_g": 35.0,
  "source_message_id": "123456789",
  "estimated_by": "item_lookup"
}
```

**body_measurement:**
```json
{
  "measured_at": "2026-04-22T10:00:00+09:00",
  "weight_kg": 70.5,
  "body_fat_percent": 15.2,
  "skeletal_muscle_mass_kg": 32.1
}
```

**routine_event:**
```json
{
  "routine_name": "clean_toilet",
  "category": "household",
  "kind": "completed",
  "note": "Used new cleaner"
}
```

**nutrition_item:**
```json
{
  "calories_kcal": 250.0,
  "protein_g": 20.0,
  "source": "user_registered"
}
```

**nutrition_target:**
```json
{
  "calories_kcal": 2400.0,
  "protein_g": 140.0,
  "effective_from": "2026-04-01",
  "effective_until": null,
  "rationale": "Lean bulk phase"
}
```

## Migration Strategy

### Data Migration

All existing specialized tables are migrated into the unified entries table:

1. `routine_events` → type `routine_event`
2. `body_measurements` → type `body_measurement`
3. `body_measurement_goals` → type `body_measurement_goal`
4. `nutrition_items` → type `nutrition_item`
5. `nutrition_logs` → type `nutrition_log`
6. `nutrition_targets` → type `nutrition_target`
7. Existing `entries` data remains as-is

After migration, the old tables are dropped.

### Code Changes

1. **DB schema** — New Atlas HCL + migration file
2. **CLI (`alt-db`)** — Remove `routine`, `nutrition-item`, `nutrition-log`, `nutrition-target` subcommands. All operations go through `alt-db entry` with appropriate `--type`
3. **Skills** — Update all skills that reference specialized CLI commands to use `alt-db entry`
4. **Webapp** — Update queries from specialized table access to entries + type filtering
5. **README** — Add "Why a single table?" section explaining the design philosophy

## README Section Draft

The following section will be added to README.md under Architecture:

### Why a Single Table?

Most applications use purpose-built tables for each domain. alt takes a different approach: one universal `entries` table for all data.

This works because alt's "application layer" is an LLM, not compiled code. Claude Code interprets metadata flexibly based on skill instructions — it doesn't need column types to function correctly. The skill definition IS the schema.

Benefits for forkers:
- **Customize by editing skills alone** — no migrations, no schema changes
- **One table to set up** — minimal Neon configuration to get started
- **Add new data types freely** — just write a new skill with a new `type` value
