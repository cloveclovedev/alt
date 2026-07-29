---
name: gh-manage-alt
description: Manage GitHub issues, labels, branches, commits, pushes, pull requests, checks, merges, and post-merge cleanup for cloveclovedev/alt using gh and local git. Use whenever Codex is asked to create, update, inspect, or close an issue; commit or push alt changes; create, update, inspect, or merge an alt pull request; inspect checks; or clean up an alt GitHub workflow.
---

# Manage alt with gh

Use this workflow only for `cloveclovedev/alt`.

## Tool policy

- Use `gh` for all GitHub-visible reads and mutations, including issues,
  labels, pull requests, checks, reviews, merges, and remote branch cleanup.
- Do not use GitHub MCP tools, apps, or connectors.
- Use local `git` for worktree inspection, branches, staging, commits, and
  local branch cleanup.
- Do not probe or use SSH from Codex.
- Do not change `origin` or run `gh auth setup-git`.
- Preserve unrelated user changes and never read ignored or sensitive files.

Before a GitHub-visible mutation, resolve and state the exact repository,
issue, pull request, label, or branch target.

## Issues and labels

Follow the repository's `AGENTS.md` rules for issue classification, labels,
body structure, priorities, dependencies, and design workflows.

Use `gh issue`, `gh label`, and `gh api` for issue operations. Write substantial
issue bodies to an exact temporary Markdown file and pass it with
`--body-file`. Remove temporary files after the command succeeds.

After creating or updating an issue, inspect it with `gh issue view` and report
its number, labels, state, and URL.

## Prepare a change

1. Confirm the repository and inspect the current branch.
2. If the branch is `main` or `master`, obtain approval for the exact new
   branch name before creating it.
3. Inspect tracked changes without reading ignored or sensitive files.
4. Stage only explicit paths in scope. Never use `git add -A` or `git add .` in
   a mixed worktree.
5. Run checks appropriate to the changed files.
6. Commit with an English Conventional Commit message.

## Push

Verify `gh auth status`, then push through gh-authenticated HTTPS without
changing the user's Git configuration:

```bash
git -c credential.helper= \
  -c 'credential.helper=!gh auth git-credential' \
  push -u https://github.com/cloveclovedev/alt.git <branch>
```

## Create or update a pull request

Use `gh pr` for all pull request operations.

1. Create or update the pull request with an English Conventional Commit title.
2. Write a substantial body to a temporary Markdown file and use
   `--body-file`.
3. Link the tracking issue with a closing keyword when the pull request
   completes it.
4. Inspect the final metadata with `gh pr view`.
5. Inspect checks with `gh pr checks`.
6. Treat failing checks as blockers unless the user explicitly waives that
   exact check.

Do not merge without explicit authorization for the target pull request.

## Squash merge

Immediately before an authorized merge:

1. Verify that the pull request is open and mergeable with `gh pr view`.
2. Re-run `gh pr checks`.
3. Stop on any failing check.
4. Record the head branch name.
5. Squash merge and request remote branch deletion:

```bash
gh pr merge <number> --repo cloveclovedev/alt --squash --delete-branch
```

6. Verify `state`, `mergedAt`, and `mergeCommit` with `gh pr view`.

## Post-merge cleanup

After a verified squash merge, always finish the workflow:

1. Switch to `main`.
2. Pull the merged `main` through gh-authenticated HTTPS:

   ```bash
   git -c credential.helper= \
     -c 'credential.helper=!gh auth git-credential' \
     pull --ff-only https://github.com/cloveclovedev/alt.git main
   ```

3. Confirm the remote head branch was deleted. If it remains, delete that exact
   branch with `gh api`; never use a broad ref or pattern.
4. Delete the exact local head branch after confirming the pull request is
   merged. A squash merge may require `git branch -D`.
5. Remove only temporary files created by this workflow.
6. Confirm that `main` points at the verified merge commit and report any
   unrelated worktree changes without modifying them.

If unrelated changes block switching or pulling, do not stash, reset, delete,
or overwrite them. Report the blocker and leave the merged pull request and
branches in their verified state.

## Report

Report the affected issue or pull request URLs, branch, source commit, checks,
merge commit when present, cleanup result, and any local or remote branch that
remains.
