# alt OSS Release Design

## Context

alt is a personal "second brain" system built on Claude Code skills, Python CLI tools,
a Next.js dashboard, and Neon Postgres. It integrates with Discord, Google Calendar,
GitHub Issues, Home Assistant, and X (Twitter) to automate daily/weekly planning,
routine tracking, nutrition monitoring, and content drafting.

The repository is currently private (`mkuri/alt`). Making it public serves two goals:

1. **Reference architecture** — Demonstrate how to wire Claude Code (skills, cloud
   triggers, rules) with external services (Neon, Discord, Home Assistant) for
   personal life management. The lower-layer integration patterns are the primary
   value for other developers.
2. **Marketing funnel** — Public repo + X posts + articles drive awareness to
   CloveClove and PepperCheck. This genre of "AI-assisted life management" content
   performs well.

The project is NOT intended to become a general-purpose framework. Individual
customization is the nature of this kind of system; the value is in showing the
patterns, not providing a one-size-fits-all solution.

Related: [GitHub Issue #71](https://github.com/mkuri/alt/issues/71)

## Decision

Publish a cleaned version of the repository under `cloveclovedev/alt` (public), with
sensitive data removed and example configs provided. Maintain `mkuri/alt` (private)
as the personal instance, tracking upstream.

## Repository Strategy

### Public repo: `cloveclovedev/alt`

- Fresh git history (no old commits that may contain secrets)
- All config files with real values replaced by `.example` templates
- MIT license
- README oriented toward developers exploring Claude Code integration patterns

### Private repo: `mkuri/alt`

- Created by cloning `cloveclovedev/alt`, then set as a separate private repo
- Remotes: `origin` → `mkuri/alt`, `upstream` → `cloveclovedev/alt`
- Personal config (`.env`, `alt.toml`) kept as untracked local files (gitignored)
- Generic improvements pushed to upstream via PRs; personal changes stay local

### Config leak prevention

Risk: accidentally including personal config in PRs to the public repo.

Mitigation strategy (details to be designed in a separate issue):
- `alt.toml` and `.env` must remain gitignored in both repos
- Personal config backed up outside of git (e.g., dotfiles, encrypted store)
- Pre-push hook or PR template checklist to catch accidental inclusion
- Workflow guidance documented in CONTRIBUTING.md or repo README

## Files to Publish

| Component | Include | Notes |
|---|---|---|
| `.claude/skills/` (13 skills) | Yes | Core content — planning, routines, cloud scheduler, X automation |
| `.claude/rules/` | Yes | Operational rules as reference |
| `src/` (alt_db, alt_discord, alt_body, alt_home_assistant) | Yes | CLI modules |
| `webapp/` (Next.js dashboard) | Yes | Part of operational flow (e.g., X draft approval) |
| `db/` (Atlas migrations) | Yes | DB schema reference |
| `tests/` | Yes | Test examples for CLI modules |
| `docs/superpowers/specs/` | Yes | Design documents as decision records |
| `docs/superpowers/plans/` | No | Working documents, excluded per plan-document-policy |
| `data/routines/*.yml` | No | Being migrated to DB; no longer relevant |
| `CLAUDE.md` | Yes | Project description |
| `pyproject.toml`, `uv.lock` | Yes | Dependency definitions |

### Config file handling

| File | Public repo | Notes |
|---|---|---|
| `.env` | `.env.example` only | Already exists; verify it covers all required vars |
| `alt.toml` | `alt.toml.example` only | Webhook URLs, channel IDs, calendar context → placeholders |
| `webapp/.env.local` | Excluded (gitignored) | Add `webapp/.env.local.example` if not present |

## README Structure (Phase A)

1. **What is alt** — 1-2 paragraph overview of the system and its purpose
2. **Architecture Overview** — Text diagram showing Claude Code Cloud / Neon / Discord / Home Assistant / GitHub Issues / Google Calendar integration
3. **Components** — Table of skills, CLI modules, and webapp with brief descriptions
4. **Getting Started** — Copy `.example` files, `uv sync`, basic setup steps
5. **License** — MIT

## Phased Rollout

### Phase A: Public Release (Issue #71)

- Audit and remove sensitive data from codebase
- Create `alt.toml.example` with placeholder values
- Rewrite README for public audience
- Add MIT LICENSE
- Create `cloveclovedev/alt` with clean initial commit
- Set up `mkuri/alt` private instance with upstream tracking
- Implement config leak prevention measures

### Phase B: User-Facing Site (Separate issue, P1)

- Landing page / documentation site (under cloveclove.dev or standalone)
- Detailed setup guide with prerequisites and step-by-step instructions
- Per-skill and per-CLI-module documentation
- Visual architecture diagram
- Potentially a blog post / Zenn article as launch content

## Out of Scope

- Generalizing skills into a reusable framework (YAGNI — personal customization is the point)
- Extracting `alt-core` as an installable package (revisit if demand materializes)
- Accepting contributions that push toward lowest-common-denominator features
- Maintaining backward compatibility for external consumers
