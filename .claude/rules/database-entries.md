# Database Entries Management

## Entry Types Overview

Use `uv run alt-db entry` to manage entries in Neon Postgres (the second brain store). All persistent data is stored as rows in the `entries` table with a `type` field that determines schema conventions.

| Type | Purpose |
|---|---|
| `task` | One-off action items with optional priority and due date |
| `knowledge` | Decisions, standards, reference info to persist |
| `goal` | Trackable objectives with target dates |
| `memo` | Temporary notes, ideas, observations |
| `tech_interest` | Technologies to explore or evaluate |
| `business` | Business-related decisions and plans |
| `routine_event` | Routine completion / baseline records |
| `body_measurement` | Body composition snapshots (e.g., InBody) |
| `body_measurement_goal` | Body composition target settings |
| `nutrition_item` | Food items registry (calorie/protein per item) |
| `nutrition_log` | Daily food intake records |
| `nutrition_target` | Daily calorie/protein targets |
| `nutrition_thread` | Discord thread tracking for daily nutrition logging |
| `daily_plan` | Posted daily plan archive |
| `weekly_plan` | Posted weekly plan archive |
| `x_draft` | X (Twitter) post drafts awaiting approval |
| `wake_event` | Wake-up nudge attempts |

## Per-Type Reference

Each section follows the same shape:
- Status: allowed values (with default if any)
- Metadata: schema for the JSON metadata column
- Content: how the content column is used
- Parent: how parent_id is used
- Title: format convention (include only when the title format is load-bearing — e.g., used as a query key, a routine name match, or a date-encoded identifier)
- Lifecycle: how entries of this type come into being and transition
- Consumers: skills/UI that read this type

### task

- Status: active(default) / backlog / done / cancelled
- Metadata: { priority: "P0|P1|P2|P3" (optional), due_date: "YYYY-MM-DD" (optional) }
- Content: free-form notes (optional)
- Parent: optional — references parent task entry id for sub-tasks; parent and child statuses are independent
- Lifecycle: Discord memo → daily-plan/weekly-plan promotion → active or backlog → done or cancelled
- Consumers: daily-plan (active list + notable elevation to summary), weekly-plan (active list + backlog review), webapp `/entries`

### knowledge

- Status: not used
- Metadata: not used
- Content: full body of the knowledge entry (decisions, standards, design principles)
- Parent: not used
- Lifecycle: created manually when a decision should persist for future reference
- Consumers: weekly-plan (recent entries review), webapp `/entries`

### goal

- Status: active / backlog / achieved
- Metadata: { target_date: "YYYY-MM" }
- Content: goal description
- Parent: not used
- Lifecycle: created manually; reviewed in weekly-plan; transitions to achieved when met
- Consumers: daily-plan (active goals, due-soon flag), weekly-plan (active + stale review), webapp `/entries`

### memo

- Status: not used
- Metadata: not used
- Content: free-form text
- Parent: not used
- Lifecycle: quick capture for ideas or observations; reviewed during weekly-plan or routinely promoted into other types if they prove worth tracking
- Consumers: daily-plan (recent memos), weekly-plan (recent memos), webapp `/entries`

### tech_interest

- Status: not used (typically)
- Metadata: not used (typically)
- Content: technology name + brief context
- Parent: not used
- Lifecycle: captured when a technology to explore comes up; reviewed periodically
- Consumers: webapp `/entries`

### business

- Status: not used (typically)
- Metadata: not used (typically)
- Content: business decision, plan, or note
- Parent: not used
- Lifecycle: captured when a business-related decision should persist
- Consumers: webapp `/entries`

### routine_event

- Status: completed / baseline
- Metadata: { category: string, completed_at: "ISO8601 with offset" }
- Content: optional note about the completion (e.g., next appointment date)
- Parent: not used
- Title: routine name; must match a key in `config.routines` (the routines skill matches completion records to definitions by title)
- Lifecycle: created when a routine is performed (`completed`) or when establishing a starting point for a new routine (`baseline`)
- Consumers: routines skill (overdue/due-soon calculation), daily-plan (overdue routines), weekly-plan (week's routines)

### body_measurement

- Status: not used (time-series data)
- Metadata: { measured_at, weight_kg, skeletal_muscle_mass_kg, muscle_mass_kg, body_fat_mass_kg, body_fat_percent, bmi, basal_metabolic_rate, inbody_score, waist_hip_ratio, visceral_fat_level, ffmi, skeletal_muscle_ratio }
- Content: not used
- Parent: not used
- Title: `InBody YYYY-MM-DD`
- Lifecycle: imported from InBody export via `alt-body` CLI; deduplicated by `metadata.measured_at`
- Consumers: webapp `/body` page (charts and history)

### body_measurement_goal

- Status: active / achieved
- Metadata: { metric: string, target_value: number, start_value: number|null, start_date: "YYYY-MM-DD", target_date: "YYYY-MM-DD" }
- Content: not used
- Parent: not used
- Title: metric name (e.g., `weight_kg`)
- Lifecycle: created via webapp body goal UI; status flips to achieved when target met
- Consumers: webapp `/body` page

### nutrition_item

- Status: not used
- Metadata: { calories_kcal: number, protein_g: number, source: "user_registered" | … }
- Content: not used
- Parent: not used
- Title: food/item name
- Lifecycle: registered via Discord meal-log workflow when a recurring food item is identified
- Consumers: nutrition-check-cloud skill (lookup table for known foods)

### nutrition_log

- Status: not used
- Metadata: { logged_date: "YYYY-MM-DD", meal_type: "breakfast|lunch|dinner|snack|supplement", calories_kcal: number, protein_g: number, source_message_id: string, estimated_by: "item_lookup" | "label_read" | "web_lookup" | "llm" }
- Content: not used (sometimes notes)
- Parent: not used
- Title: food description as logged
- Lifecycle: created by nutrition-check-cloud skill from Discord meal thread messages
- Consumers: nutrition-check-cloud skill (daily totals), future nutrition dashboard

### nutrition_target

- Status: active
- Metadata: { calories_kcal: number, protein_g: number }
- Content: not used
- Parent: not used
- Title: target descriptor (e.g., `daily`)
- Lifecycle: created/updated via CLI when targets change; only one active at a time
- Consumers: nutrition-check-cloud skill (target comparison)

### nutrition_thread

- Status: not used
- Metadata: { date: "YYYY-MM-DD", thread_id: string }
- Content: thread_id (also stored in metadata)
- Parent: not used
- Title: `Nutrition Thread <date>`
- Lifecycle: created by nutrition-check-cloud when a daily meal thread is opened in Discord
- Consumers: nutrition-check-cloud skill (locating today's thread)

### daily_plan

- Status: posted
- Metadata: { source: "local" | "cloud", thread_id: string }
- Content: full plan text (summary + thread detail combined with `---` separator)
- Parent: not used
- Title: `Daily Plan YYYY-MM-DD`
- Lifecycle: created at the end of daily-plan or daily-plan-cloud after Discord posting
- Consumers: daily-plan-cloud (idempotency check), wake-check-cloud (presence check), weekly-plan (week's history)

### weekly_plan

- Status: posted
- Metadata: not used
- Content: full plan text
- Parent: not used
- Title: `Weekly Plan <monday YYYY-MM-DD>`
- Lifecycle: created at the end of weekly-plan or weekly-plan-cloud after Discord posting
- Consumers: weekly-plan-cloud (idempotency check)

### x_draft

- Status: draft / approved / posted
- Metadata: { source_commits, source_memo_count, source_design_doc, source_pr_url, generated_at, image_url, post_type: "progress|technical|problem-solution|reflection", hashtags, reply_link, reply_link_label, project }
- Content: draft post text
- Parent: not used
- Title: one-line summary
- Lifecycle: created by x-draft-cloud from recent activity; reviewed in webapp `/posts`; status flips to `approved` upon human review; consumed by x-post-cloud when posting
- Consumers: webapp `/posts` (review/approve), x-post-cloud (post when due)

### wake_event

- Status: sent
- Metadata: { attempt: number, scenario: "morning" | "night", target_time: "HH:MM" }
- Content: message text sent
- Parent: not used
- Title: `Wake: <scenario> attempt <N>`
- Lifecycle: created by wake-check-cloud each time a nudge is sent; idempotency tracking
- Consumers: wake-check-cloud (escalation logic, idempotency)

## CLI Commands

```bash
# Add
uv run alt-db entry add --type <type> --title "Title" --content "Content"

# With status and metadata (e.g., goals)
uv run alt-db entry add --type goal --title "Title" --status active --metadata '{"target_date":"2026-09"}'

# List / Search
uv run alt-db entry list --type <type>
uv run alt-db entry list --type goal --status active
uv run alt-db entry list --since 7d
uv run alt-db entry list --due-within 7d
uv run alt-db entry search "keyword"

# Update / Delete
uv run alt-db entry update <id> --status achieved
uv run alt-db entry delete <id>
```

## When to Save

Save an entry when information is:
- A **decision** that should inform future work (tech stack, conventions, policies)
- A **goal or objective** with a clear outcome
- A **task** to act on (one-off, optionally with priority and due date)
- A **note or idea** worth revisiting later
- **Not derivable** from code, git history, or existing docs

Do NOT save:
- Code patterns derivable from the codebase
- Information already in CLAUDE.md or rules files

## Language

All entry content MUST be written in English — title and content. This aligns with the GitHub conventions rule (English for all persistent content).

## Content Guidelines

- Include structured summary, not raw dumps
- For long documents, summarize key decisions and rationale in the entry content
