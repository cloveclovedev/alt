# Daily Run Design

- Status: Draft
- Intended delivery: a separate pull request after daily planning
- Depends on:
  - `routine.md`
  - `daily-planning.md`

## Why this document is a draft

The daily run should be refined after the confirmed-plan experience is used in
practice. This document records the decisions already made and preserves the
direction of the feature without prematurely fixing every interaction or table.

## Context

The guided run in `social_manager` removes repeated "what should I do next?"
decisions by presenting a fixed, scoped queue and requiring each item to reach a
terminal state before advancing. The useful principle is subtractive: preserve
the information required to act while removing repeated ranking and navigation.

Alt has a different mix of work. GitHub issues often benefit from a sequence,
but routines may be performed concurrently: a washing machine can run while a
room is cleaned. Calendar events are usually constraints, not actionable cards.
A strict one-item queue would therefore impose the wrong execution model.

## Goals

1. Start from one confirmed daily-plan revision.
2. Preserve the plan's category and priority decisions.
3. Reduce repeated choice during execution.
4. Let routine items be completed in any order within their category.
5. Resume safely after leaving or refreshing the page.
6. Keep GitHub as the sole source of truth for issue lifecycle.
7. Reuse the routine completion service rather than duplicating routine state.

## Non-goals

- Closing, editing, or commenting on GitHub issues.
- Treating Calendar events as tasks.
- Automatically browser-driving development work.
- A generalized workflow engine.
- Final analytics or productivity scoring.
- Freezing the interaction before real daily-plan usage provides evidence.

## Entry point

The home page shows the current confirmed daily plan and a "Start run" action.
Starting a run captures the current `plan_revision_id`. A run never silently
switches to a later revision.

The entry view groups current items by category and shows their planned order.
The first implementation may allow removal before start, but does not need
automatic AI grouping beyond the categories already present in the plan.

## Category behavior

### GitHub issues

GitHub issues are shown in the confirmed plan order, one at a time.

Terminal actions are:

- `Done for today`: records that the planned work item was handled in this run;
- `Remove from today`: removes it from the remainder of this run.

Neither action changes the issue on GitHub. The next planning session fetches
GitHub again; a closed issue naturally disappears, while an open issue remains
eligible. "Done for today" is run progress, not a duplicate issue status.

If an issue is observed as closed while a run is active, the UI may mark it
externally resolved and advance after user acknowledgment.

### Routines

Routines are grouped by their user-managed routine category. Every routine in
the current category remains visible as a checklist, and completions may occur
in any order.

Terminal actions are:

- `Complete`: calls the routine service and creates a real completion event;
- `Remove from today`: removes the routine from this run without changing its
  due date or active status.

This supports parallel real-world activity without inventing an automatic
parallel-execution graph.

### Action items

Plan-local action items may use:

- `Done for today`;
- `Remove from today`.

They have no independent backlog after the run because they are not tasks.
Whether an unfinished action item should be promoted into a future task is
deferred until task management is designed.

### Calendar events

Calendar events appear in a schedule/context area:

- current time;
- current or next commitment;
- available time until the next commitment.

They do not require terminal actions and do not participate in run completion.

## Persistence direction

The run must be resumable, so progress belongs in PostgreSQL rather than URL
parameters alone.

A likely model is:

```text
daily_runs
id
user_id
plan_revision_id
status
started_at
completed_at
updated_at
```

Item progress must reference the stable UUIDs of structured plan-revision
components. The exact representation remains open until the first daily-plan
schema is implemented and exercised. It should preserve real foreign keys and
avoid an unchecked polymorphic `(type, id)` pair if practical.

## Known state semantics

Run-level states are expected to include:

- `active`;
- `completed`;
- `abandoned`.

Leaving the page does not abandon a run. It remains active and can be resumed.

Item-level semantics distinguish:

- pending;
- handled for today;
- removed from today;
- externally resolved, for a GitHub issue observed closed.

Routine completion remains in the routine domain; the run stores only the fact
that its corresponding plan item reached a terminal state.

## Revision changes during a run

An active run remains tied to the revision from which it started. A later plan
revision does not rewrite its selected items.

The product behavior when the user confirms a new revision while a run is
active is intentionally unresolved. Candidate behaviors are:

1. finish or abandon the old run before starting the new revision;
2. explicitly migrate unresolved items into a new run;
3. keep both revisions available and let the user choose.

This should be decided from actual usage rather than hidden automatic merging.

## Completion experience

Completing all actionable items reaches a dedicated completion state. Calendar
events do not block completion.

The completion screen should be small and useful:

- show handled, removed, and externally resolved counts;
- offer return to the current plan;
- avoid adding scoring or analytics before a concrete need appears.

## Initial testing direction

- A run is tied to exactly one confirmed revision.
- Refresh and navigation preserve active progress.
- GitHub "Done for today" never calls a GitHub mutation.
- A closed GitHub issue can be recognized without duplicating its lifecycle.
- Routine completion creates exactly one routine event and terminates the run
  item atomically.
- Routine checklist completion order is unrestricted.
- Removing a routine from today does not alter its due calculation.
- Calendar events never block run completion.
- Completed and abandoned runs cannot accept new progress.

## Open product questions

- Whether users select a subset of plan items before starting.
- Exact category order and whether it is editable at run start.
- Whether `Remove from today` requires a reason.
- How a newly confirmed revision interacts with an active older run.
- Whether action items need a future carry-over flow after task management is
  designed.
- The final normalized persistence model for heterogeneous run items.

These questions are intentionally deferred to the daily-run pull request.

## Decision log

This is a living design document. Each entry records a change to the design;
the sections above always describe the current intended design.

- 2026-07-29 — Initial draft, to be refined after the confirmed-plan experience
  is used in practice.
