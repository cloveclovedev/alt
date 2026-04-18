---
name: routines
description: Use when checking routine tasks, viewing overdue items, or marking routines as completed
---

# Routine Management

## When invoked

1. **Verify DB connection:**
   Run: `uv run alt-db routine all` to confirm connectivity.

2. **Load routine definitions:**
   Read all YAML files in `data/routines/` to get the full list of routines with their intervals.

   Each routine has required and optional fields:
   - `name` (required): routine name
   - `interval_days` (required): days between completions
   - `active_months` (optional): array of months 1-12 when routine is active. Omit = always active.
   - `available_days` (optional): array of days (`mon`-`sun`) when routine can be actioned. Omit = any day.
   - `notes` (optional): free-text context displayed alongside the routine.

3. **Load completion history:**
   Run: `uv run alt-db routine all` to get last completion dates for all routines.

4. **Calculate overdue routines:**
   For each routine, compare `last_completed + interval_days` against today's date.
   If never completed, treat as overdue.

   Then apply filters:
   - **Active months filter**: If `active_months` is set and the current month is NOT in the list, exclude the routine entirely (not displayed). When the active season resumes and the routine has never been completed, it appears as overdue.
   - **Available days filter**: If `available_days` is set and today's day-of-week is NOT in the list, route overdue and due-soon routines to the "Overdue (not actionable today)" section.

5. **Present to user:**
   Show only sections that have items. Hide Paused (off-season) and OK routines.

   - **Overdue**: past due AND actionable today
   - **Overdue (not actionable today)**: past due or due soon, but today is not in `available_days`
   - **Due Soon**: within 3 days of due AND actionable today

   Display `notes` with "Notes:" prefix when present.

6. **Check DB notes:**
   When displaying routines, also check the latest completion record's `note` field for each routine.
   If a note exists (e.g., next appointment date, special instructions), display it alongside the routine.
   Individual routine-specific details (next appointment, scheduling preferences, deferral reasons) should be stored in the DB `note` field — not in Claude Code memory or YAML definitions.

7. **Interactive actions:**
   Ask the user if they want to mark any routines as completed.
   Users can mark routines from any displayed section, including "Overdue (not actionable today)".
   When completing a routine, always match the user's input against existing routine names from the `uv run alt-db routine all` output. Never create a new routine name — if no match is found, ask the user to clarify which routine they mean.
   For each completion, run: `uv run alt-db routine complete "<name>" "<category>" --note "<optional note>"`
   For setting a tracking baseline: `uv run alt-db routine baseline "<name>" "<category>" --date "<YYYY-MM-DD>" --note "<optional note>"`
   When the user provides context about the next occurrence (e.g., next appointment date, deferral reason), include it in the note.
   All notes MUST be written in English.

8. **Correcting mistakes:**
   When a routine is incorrectly marked as completed, or a note needs correction:
   1. Run `uv run alt-db routine history "<name>"` to list all records with IDs
   2. Run `uv run alt-db routine delete <id>` to remove incorrect records
   3. Run `uv run alt-db routine update-note <id> --note "..."` to fix notes on existing records

## Output format

```
### Overdue
- Clean the toilet (household) — 18 days since last, was due 4 days ago
- Change tooth brush (household) — 35 days since last, was due 5 days ago

### Overdue (not actionable today)
- Wash the comforter (household) — 120 days since last, was due 30 days ago
  Notes: Requires coin laundry

### Due Soon
- Wash the bed and pillow sheets (household) — 5 days since last, due in 2 days
```
