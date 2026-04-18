# Body Measurements Webapp Visualization

Display body measurement data on the webapp with goal tracking, ideal vs actual progression charts, and goal management UI.

## Context

- ~60 InBody CSV measurements imported since Aug 2025 (~weekly frequency)
- Data stored in `body_measurements` table (Neon Postgres)
- Webapp: Next.js 16 + React 19 + Tailwind v4 + shadcn/ui
- No chart library currently installed

## Data Model

### New table: `body_measurement_goals`

| Column | Type | Description |
|---|---|---|
| `id` | UUID PK | auto-generated |
| `metric` | TEXT NOT NULL | Metric name (`weight_kg`, `body_fat_percent`, `muscle_mass_kg`, `ffmi`, etc.) |
| `target_value` | DECIMAL(6,2) NOT NULL | Target value |
| `start_value` | DECIMAL(6,2) | Value at goal creation (ideal line start point) |
| `start_date` | DATE NOT NULL | Goal start date |
| `target_date` | DATE NOT NULL | Target deadline |
| `status` | TEXT NOT NULL DEFAULT 'active' | `active`, `achieved`, `expired`, `cancelled` |
| `created_at` | TIMESTAMPTZ | default now() |

- Partial unique index on `(metric)` WHERE `status = 'active'` — one active goal per metric
- Goal history preserved as rows with non-active status
- Replaces `alt.toml` `[body.goals]` section (remove from alt.toml as part of this work)
- `metric` values must match `body_measurements` column names exactly (e.g. `weight_kg`, `body_fat_percent`, `muscle_mass_kg`, `ffmi`)

### Database migration

- Add HCL schema to `db/schema/body_measurement_goals.hcl`
- Use Atlas versioned migration: `atlas migrate diff` to generate SQL
- New `db/migrations/` directory (first versioned migration in the project)
- Existing tables remain as-is (no baseline migration needed)

## Page Structure

### Dashboard (`/`)

Existing cards (Deadline Alerts, Active Goals, Recent Memos) remain unchanged.

Add **Body Composition** card:
- 2x2 grid of compact charts for 4 primary metrics: weight, body_fat_percent, muscle_mass, ffmi
- Each chart shows actual line (solid) + ideal line (dashed, if active goal exists)
- Fixed 90-day period
- Metric name + latest value displayed above each chart
- Minimal axis labels, no tooltips
- Click card or "View details" link to navigate to `/body`

### Detail page (`/body`)

1. **Period selector**: 30d / 90d / 6m / 1y / All
2. **Primary metrics charts**: Same 4 metrics as dashboard but full-size with tooltips, axis labels, grid lines
3. **Secondary metrics chart**: Dropdown to select from bmi, inbody_score, basal_metabolic_rate, skeletal_muscle_mass_kg, waist_hip_ratio, visceral_fat_level. One chart area, metric switchable.
4. **Goal setting section**: Per-metric cards with inline form for setting/editing goals
5. **Latest measurement summary**: All metrics from most recent measurement
6. **Measurement history table**: Paginated table of past measurements

### Navigation

Add "Body" link to nav component between existing links.

## Chart Specifications

### Library

shadcn/ui chart component (`npx shadcn add chart`), which wraps Recharts internally. Consistent with existing UI theming.

### Visual design

- **Color scheme**: Weight (blue), Body Fat % (pink), Muscle Mass (green), FFMI (purple)
- **Actual line**: Solid, 2px stroke
- **Ideal line**: Dashed, same color at reduced opacity. Linear interpolation from (start_date, start_value) to (target_date, target_value). Target value label at the endpoint.
- **No goal horizontal line** — the ideal line endpoint serves as the goal indicator
- **Y-axis**: Auto-scaled with padding above/below data range
- **X-axis**: Date labels, granularity adapts to period (days for 30d, weeks for 90d, months for 1y+)

### Dashboard compact variant

- No tooltips
- Minimal axis labels
- Metric name + latest value header
- Clickable (navigates to `/body`)

### Detail full variant

- Hover tooltips showing date + value
- Full axis labels and grid lines
- Period selector controls all charts

### No active goal

Ideal line and goal label hidden. Only actual line displayed.

## Component Structure

```
webapp/src/
├── app/
│   ├── page.tsx                          # Dashboard (existing + BodySummaryCard)
│   ├── body/
│   │   └── page.tsx                      # Detail page
│   └── api/body/
│       ├── measurements/route.ts         # GET measurements
│       └── goals/
│           ├── route.ts                  # GET/POST goals
│           └── [id]/route.ts             # PUT goal
├── components/
│   ├── ui/chart.tsx                      # shadcn chart (npx shadcn add chart)
│   └── body/
│       ├── body-summary-card.tsx         # Dashboard 2x2 grid
│       ├── metric-chart.tsx              # Single metric chart (compact/full via prop)
│       ├── period-selector.tsx           # Period toggle 30d/90d/6m/1y/All
│       ├── metric-selector.tsx           # Secondary metric dropdown
│       ├── goal-card.tsx                 # Goal display/edit per metric
│       ├── latest-summary.tsx            # Latest measurement all metrics
│       └── measurement-history.tsx       # Paginated history table
└── lib/
    └── body.ts                           # DB query functions
```

### Key design decisions

- `metric-chart.tsx` is the core reusable component. `compact` prop toggles between dashboard and detail variants.
- `/body` page: Server Component for initial data fetch. Period selector and goal forms are Client Components.
- API routes handle period filtering (query param) and goal CRUD.

## API Routes

### `GET /api/body/measurements?period=90d`

Returns measurements within the specified period. Period values: `30d`, `90d`, `6m`, `1y`, `all`.

### `GET /api/body/goals`

Returns all goals. Filter by `?status=active` for chart rendering.

### `POST /api/body/goals`

Create a new goal. Body: `{ metric, target_value, target_date }`. `start_value` auto-populated from latest measurement. `start_date` set to current date.

### `PUT /api/body/goals/[id]`

Update goal. Supports updating `target_value`, `target_date`, and `status`.

## Goal Setting UI

- Per-metric card layout in the goal setting section
- **Active goal exists**: Display target value, deadline, progress. Edit button opens inline form. Status change buttons (achieved/expired/cancelled).
- **No goal**: "Set goal" button expands inline form
- **Form fields**: target_value (number input), target_date (date input)
- **Past goals**: Collapsible history with status badges
- No modals — all inline

## Edge Cases

- **No measurement data**: Dashboard card and detail page show empty state with message
- **Goal period extends beyond chart period**: Ideal line clips at chart boundary, goal label still shown at endpoint
- **Ideal line in the past**: If target_date has passed and goal is still active, ideal line renders fully (visual cue that deadline passed)
- **Goal for secondary metrics**: Supported — secondary chart shows ideal line when active goal exists for selected metric
