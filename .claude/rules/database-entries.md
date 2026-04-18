# Database Entries Management

## Entry Types

Use `uv run alt-db entry` to manage entries in Neon Postgres (the second brain store).

| Type | Purpose | Example |
|---|---|---|
| `knowledge` | Decisions, standards, reference info to persist | Tech stack choices, design principles |
| `goal` | Trackable objectives with target dates | "Launch app by 2026-09" |
| `memo` | Temporary notes, ideas, observations | Meeting notes, quick ideas |
| `tech_interest` | Technologies to explore or evaluate | "Look into Deno 2.0" |
| `business` | Business-related decisions and plans | Revenue targets, strategy |

## CLI Commands

```bash
# Add
uv run alt-db entry add --type <type> --title "Title" --content "Content" --tags '["tag1", "tag2"]'

# With status and metadata (e.g., goals)
uv run alt-db entry add --type goal --title "Title" --status active --tags '["tag"]' --metadata '{"target_date":"2026-09"}'

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
- A **note or idea** worth revisiting later
- **Not derivable** from code, git history, or existing docs

Do NOT save:
- Code patterns derivable from the codebase
- Information already in CLAUDE.md or rules files

## Tag Conventions

Use lowercase, hyphenated tags. Prefer reusing existing tags over creating new ones.

Common tags: `tech-stack`, `architecture`, `infrastructure`, `business`, `process`

## Language

All entry content MUST be written in English — title, content, and tags. This aligns with the GitHub conventions rule (English for all persistent content).

## Content Guidelines

- Include structured summary, not raw dumps
- For long documents, summarize key decisions and rationale in the entry content
