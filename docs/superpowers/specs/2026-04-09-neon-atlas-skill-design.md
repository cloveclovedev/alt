# Neon + Atlas Claude Code Skill Design

## Overview

Set up a reusable Atlas database migration skill for Claude Code, with Neon-specific best practices. The skill is stored in the dotfiles repository and symlinked into projects that use Neon + Atlas.

## File Structure

```
dotfiles/claude/optional/skills/atlas/
  SKILL.md                          # Official Atlas Agent Skill (verbatim from atlasgo.io)
  references/
    schema-sources.md               # Official schema sources reference (verbatim)
    neon-practices.md               # Neon-specific best practices (custom)
```

## Per-Project Setup

Symlink from the project's `.claude/skills/` directory:

```bash
ln -s ~/projects/dotfiles/claude/optional/skills/atlas <project>/.claude/skills/atlas
```

This makes the skill available only in projects that need it, avoiding unnecessary context consumption in unrelated projects.

## Official Files (SKILL.md, schema-sources.md)

Copied verbatim from the Atlas documentation at https://atlasgo.io/guides/ai-tools/agent-skills. These files are not modified. When Atlas updates their skill content, manually update these files to stay current.

Key capabilities provided by the official skill:
- Workflow decision tree (versioned vs declarative)
- CLI command quick reference
- Security guidelines (never hardcode credentials)
- Standard migration workflow (inspect -> edit -> validate -> diff -> lint -> dry-run -> apply)
- ORM integration references
- Troubleshooting guidance

## Custom: neon-practices.md

### 1. Connection URL Composition

Compose the database URL from individual environment variables in `atlas.hcl` using `locals` and `getenv()`:

```hcl
locals {
  db_url = "postgresql://${getenv("NEON_USER")}:${getenv("NEON_PASSWORD")}@${getenv("NEON_HOST")}/${getenv("NEON_DATABASE")}?sslmode=require"
}
```

Required environment variables (defined in `.env`):
- `NEON_HOST` — Direct connection endpoint (e.g., `ep-xxx.region.aws.neon.tech`, without `-pooler`)
- `NEON_DATABASE` — Database name
- `NEON_USER` — Database user
- `NEON_PASSWORD` — Database password

No full `DATABASE_URL` environment variable is needed. The URL is composed at the atlas.hcl level to avoid duplication.

### 2. atlas.hcl Neon Template

```hcl
locals {
  db_url = "postgresql://${getenv("NEON_USER")}:${getenv("NEON_PASSWORD")}@${getenv("NEON_HOST")}/${getenv("NEON_DATABASE")}?sslmode=require"
}

env "neon" {
  src = "file://schema"
  url = local.db_url
  dev = "docker://postgres/17/dev?search_path=public"
  migration {
    dir = "file://migrations"
  }
}
```

Key points:
- `url` uses the direct Neon endpoint (not pooled) — DDL operations require direct connections to avoid timeout/lock issues with connection poolers
- `dev` uses a local Docker-based PostgreSQL — never use the Neon production database as the dev database
- `src` points to the HCL schema directory
- `migration.dir` specifies the versioned migrations directory

### 3. Operational Rules

- Always run `atlas migrate apply --env neon --dry-run` before actual apply
- Use the direct connection host (without `-pooler` suffix) for all migration operations
- Neon branching workflow details are deferred — to be designed alongside application deployment strategy

## Maintenance

- Official Atlas files (`SKILL.md`, `schema-sources.md`): check for updates when upgrading the Atlas CLI
- Custom file (`neon-practices.md`): update as Neon-specific patterns evolve

## Scope

This skill covers database schema management and migrations only. It does not cover:
- Application-level database access patterns
- Neon branching strategy for application environments (deferred)
- CI/CD pipeline integration (project-specific)
