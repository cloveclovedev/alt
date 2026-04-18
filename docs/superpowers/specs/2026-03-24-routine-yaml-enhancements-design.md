# Routine YAML Enhancements

## Problem

The current routine system treats all routines uniformly: every routine is either overdue, due soon, or OK. This creates noise because:

1. **Seasonal routines** (flea/tick medicine, filariasis) show as overdue during months they don't apply.
2. **Weekend-only routines** (comforter wash, deep cleaning) show as overdue on weekdays when they can't be actioned.
3. **Missing context** — some routines have constraints (e.g., requires coin laundry) that aren't captured anywhere.

## Solution

Add three optional fields to the routine YAML schema and update the routines skill logic to use them.

## YAML Schema Changes

Three new optional fields per routine entry:

```yaml
- name: Take flea and tick medicine
  interval_days: 30
  active_months: [4, 5, 6, 7, 8, 9, 10, 11]
  available_days: [sat, sun]
  notes: "Seasonal: Apr-Nov only"
```

### Field Definitions

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `active_months` | `int[]` (1-12) | all months | Months when the routine is active. Outside these months, the routine is paused and hidden from output. |
| `available_days` | `string[]` (`mon`-`sun`) | all days | Days of the week when the routine can be actioned. On other days, overdue items appear in a separate deferred section. |
| `notes` | `string` | none | Free-text context displayed alongside the routine. English. |

All fields are optional. Omitting them preserves current behavior.

## Routines Skill Logic Changes

### Step 4: Calculate overdue routines (updated)

The existing logic (`last_completed + interval_days` vs today) remains unchanged. Two filters are added:

1. **Active months filter**: If `active_months` is set and the current month is NOT in the list, the routine is excluded from all output sections (not displayed). When the active season resumes and the routine has never been completed, it appears as overdue — this is intentional to prompt first-time completion.
2. **Available days filter**: If `available_days` is set and today's day-of-week is NOT in the list, overdue and due-soon routines are routed to a separate "Overdue (not actionable today)" section. Due-soon items with `available_days` restriction on non-matching days are also routed to this section (not hidden).

### Step 5: Present to user (updated output format)

Only actionable sections are displayed. Paused and OK routines are hidden.

```
### Overdue
- Clean the toilet (household) — 23 days since last, was due 9 days ago
- Clean the kitchen drain (household) — 23 days since last, was due 9 days ago

### Overdue (not actionable today)
- Wash the comforter (household) — 357 days since last, was due 267 days ago
  Notes: Requires coin laundry, weekend only

### Due Soon
- Wash the bed and pillow sheets (household) — 5 days since last, due in 2 days
```

Display rules:
- **Overdue**: past due AND actionable today (no `available_days` restriction, or today matches)
- **Overdue (not actionable today)**: past due or due soon, but today is not in `available_days`. This section label is generic to support any `available_days` combination, not just weekends.
- **Due Soon**: within 3 days of due AND actionable today (no `available_days` restriction, or today matches)
- **Paused (off-season)**: hidden (not displayed)
- **OK**: hidden (not displayed)
- `notes` field displayed with "Notes:" prefix when present on any shown routine

### Step 6: Interactive actions (unchanged but clarified)

Users can mark routines as completed from any displayed section, including "Overdue (not actionable today)". The completion flow (`./scripts/db.sh complete`) is unchanged.

### Daily-plan integration

The daily-plan skill calls the routines skill in Phase 1. The same display rules apply — only Overdue, Overdue (not actionable today), and Due Soon sections appear in the daily plan output. The daily-plan skill file (`.claude/skills/daily-plan/skill.md`) does not need changes because it delegates to the routines skill logic described here.

## Specific Routine Changes

### `data/routines/dog.yml`

| Routine | New fields |
|---------|-----------|
| Take flea and tick medicine | `active_months: [4, 5, 6, 7, 8, 9, 10, 11]`, `notes: "Seasonal: Apr-Nov only"` |
| Take medicine for filariasis | `active_months: [4, 5, 6, 7, 8, 9, 10, 11]`, `notes: "Seasonal: Apr-Nov only"` |
| Clean the dog blankets | No changes |
| Change a filter of the waterer | No changes |

### `data/routines/household.yml`

| Routine | New fields |
|---------|-----------|
| Wash the comforter | `available_days: [sat, sun]`, `notes: "Requires coin laundry"` |
| Clean the humidifier (Filters and trays) | `available_days: [sat, sun]` |
| Clean the humidifier (Main units) | `available_days: [sat, sun]` |

All other household routines: no changes (actionable any day via the Saturday cleaning block or ad-hoc).

### `data/routines/health.yml`

No changes.

## Files Changed

| File | Change |
|------|--------|
| `data/routines/dog.yml` | Add `active_months`, `notes` to 2 seasonal routines |
| `data/routines/household.yml` | Add `available_days`, `notes` to 3 weekend-only routines |
| `data/routines/health.yml` | No changes |
| `.claude/skills/routines/skill.md` | Update steps 4-5 with new filter logic and output format |

## No DB/Script Changes

`scripts/db.sh` and the SQLite schema (`routine_completions` table) remain unchanged. All new logic lives in YAML definitions and the skill markdown.
