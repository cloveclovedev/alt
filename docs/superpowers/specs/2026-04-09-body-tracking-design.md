# Body Tracking Design

## Overview

Track body composition metrics from InBody Dial measurements. Data is imported via CSV export from the InBody app, parsed and enriched with calculated metrics (FFMI, skeletal muscle ratio), and stored in a dedicated Postgres table for trend analysis and dashboard visualization.

## Context

- InBody Dial (H30) measures body composition and syncs to the InBody app
- The InBody app exports CSV files with timestamped measurements
- Google Fit REST API is deprecated (shut down June 2025); Health Connect has no REST API
- CSV import is the practical data ingestion path

## Architecture

```
InBody App → CSV export → alt-body import CLI → Neon Postgres
                              │
                              ├── parser.py   (CSV → structured data)
                              ├── metrics.py  (FFMI, skeletal muscle ratio)
                              └── storage.py  (UPSERT to body_measurements)
```

`alt-body` is a separate Python package from `alt-db`. It owns the domain logic (parsing, calculation) and uses `alt-db`'s connection module for database access.

## Database Schema

New table `body_measurements` managed via Atlas HCL.

```sql
CREATE TABLE body_measurements (
  id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  measured_at              TIMESTAMPTZ NOT NULL UNIQUE,
  -- Raw measurements (from InBody CSV)
  weight_kg                DECIMAL(5,2) NOT NULL,
  skeletal_muscle_mass_kg  DECIMAL(5,2),
  muscle_mass_kg           DECIMAL(5,2),
  body_fat_mass_kg         DECIMAL(5,2),
  body_fat_percent         DECIMAL(4,1),
  bmi                      DECIMAL(4,1),
  basal_metabolic_rate     INTEGER,
  inbody_score             DECIMAL(4,1),
  waist_hip_ratio          DECIMAL(3,2),
  visceral_fat_level       INTEGER,
  -- Calculated metrics
  ffmi                     DECIMAL(4,2),
  skeletal_muscle_ratio    DECIMAL(4,1),
  -- Metadata
  source                   TEXT NOT NULL DEFAULT 'inbody_csv',
  created_at               TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_body_measurements_measured_at
  ON body_measurements(measured_at DESC);
```

**Design decisions:**
- `measured_at UNIQUE` prevents duplicate imports (same CSV can be imported multiple times safely)
- Timestamp is second-precision from InBody CSV (format: `YYYYMMDDHHmmss`), so multiple measurements per day are stored as separate records
- Raw measurements and calculated metrics are in separate column groups for clarity
- `source` column enables future data sources beyond InBody CSV

## Package Structure

```
src/alt_body/
├── __init__.py
├── cli.py          # CLI entry point
├── parser.py       # InBody CSV parsing
├── metrics.py      # FFMI, skeletal muscle ratio calculation
└── storage.py      # DB write operations (UPSERT)
```

Entry point registered in `pyproject.toml`:
```toml
[project.scripts]
alt-body = "alt_body.cli:main"
```

## CLI Interface

```bash
uv run alt-body import /path/to/InBody-20260409.csv
```

Output:
```
Imported 5 new measurements (skipped 44 duplicates)
Latest: 2026-04-03 03:45 — 64.9kg / BF 21.0% / FFMI 20.12
```

## CSV Parsing

InBody CSV format (Japanese headers):
- `日付`: `YYYYMMDDHHmmss` → parsed to `TIMESTAMPTZ` (Asia/Tokyo)
- Numeric fields: parsed as float, `-` values mapped to `NULL`
- Header row is skipped

Columns mapped:
| CSV Header | DB Column |
|---|---|
| 日付 | measured_at |
| 体重(kg) | weight_kg |
| 骨格筋量(kg) | skeletal_muscle_mass_kg |
| 筋肉量(kg) | muscle_mass_kg |
| 体脂肪量(kg) | body_fat_mass_kg |
| 体脂肪率(%) | body_fat_percent |
| BMI(kg/m²) | bmi |
| 基礎代謝量(kcal) | basal_metabolic_rate |
| InBody点数 | inbody_score |
| ウエストヒップ比 | waist_hip_ratio |
| 内臓脂肪レベル(Level) | visceral_fat_level |

## Metric Calculations

Height is read from `alt.toml`:
```toml
[body]
height_m = 1.73

[body.goals]
ffmi = 21.5
body_fat_percent = 18.0
```

### FFMI (Fat-Free Mass Index)

```python
lbm = weight_kg * (1 - body_fat_percent / 100)
ffmi_raw = lbm / (height_m ** 2)
ffmi = ffmi_raw + 6.1 * (1.8 - height_m)  # height-normalized
```

Interpretation: 20+ = sporty, 22+ = athlete-level.

### Skeletal Muscle Ratio

```python
skeletal_muscle_ratio = (skeletal_muscle_mass_kg / weight_kg) * 100
```

Uses actual measured skeletal muscle mass from InBody (not estimated).

## Storage Logic

- Uses `INSERT ... ON CONFLICT (measured_at) DO NOTHING` for idempotent imports
- Shares `alt_db.connection` module for Neon HTTP API access
- Returns count of new inserts and skipped duplicates

## Configuration

Added to `alt.toml`:
```toml
[body]
height_m = 1.73

[body.goals]
ffmi = 21.5
body_fat_percent = 18.0
```

Goals are stored in config (not DB) since they change infrequently and are referenced by dashboard and daily-plan.

## Scope

**This PR:**
- `body_measurements` table (Atlas HCL schema)
- `alt-body` Python package (parser, metrics, storage, CLI)
- `alt.toml` body section
- Tests

**Future PRs:**
- Webapp dashboard (trend charts, goal lines)
- Daily-plan integration (latest metrics, trend summary, goal gap reminder)
