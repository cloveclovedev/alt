---
name: codex-publish-alt-pr
description: Publish changes from the cloveclovedev/alt repository with Codex by creating a scoped branch, committing intentional paths, pushing through gh-authenticated HTTPS, opening a pull request, checking its status, and squash-merging when authorized. Use when the user asks Codex to commit, push, open or update a PR, inspect PR checks, or merge changes in alt.
---

# Publish an alt Pull Request

Follow this workflow only in `cloveclovedev/alt` and only when operating as
Codex. Keep repository-wide policy separate from this client-specific
authentication procedure.

## Prepare the change

1. Confirm the repository and inspect the current branch.
2. If the branch is `main` or `master`, obtain approval for the exact new branch
   name before creating it.
3. Inspect tracked changes without reading ignored or sensitive files.
4. Stage only the explicit paths in scope. Never use `git add -A` or
   `git add .` in a mixed worktree.
5. Run checks appropriate to the changed files.
6. Commit with an English Conventional Commit message.

Preserve unrelated user changes throughout the workflow.

## Push from Codex

Do not probe or use SSH from Codex. Do not change `origin` and do not run
`gh auth setup-git`, because Codex may not be able to update the user's global
Git configuration.

Verify `gh` is available and authenticated, then push with a one-command
credential helper:

```bash
git -c credential.helper= \
  -c 'credential.helper=!gh auth git-credential' \
  push -u https://github.com/cloveclovedev/alt.git <branch>
```

This HTTPS rule is Codex-specific. Do not modify or disable a working SSH
workflow used by Claude Code or another client.

## Open and inspect the pull request

Use `gh` for GitHub-visible operations:

1. Create or update the PR with an English Conventional Commit title.
2. Write the PR body as real Markdown in a temporary file when shell quoting
   would make inline text fragile.
3. Inspect checks with `gh pr checks`.
4. Treat failing checks as blockers unless the user explicitly waives that
   exact check. Do not encode temporary check exceptions in this skill.
5. Merge only when the user has explicitly authorized merging this PR. Use
   squash merge unless the user requests another strategy.
6. Verify the final PR state and merge commit with `gh pr view`.

Prefer commands shaped like:

```bash
gh pr create --repo cloveclovedev/alt --title "<title>" --body-file <file>
gh pr checks <number> --repo cloveclovedev/alt
gh pr merge <number> --repo cloveclovedev/alt --squash
gh pr view <number> --repo cloveclovedev/alt \
  --json state,mergedAt,mergeCommit,url
```

## Report the result

Report the branch, commit, PR URL, check result, and merge commit when present.
Also state whether any local or remote branch remains.
