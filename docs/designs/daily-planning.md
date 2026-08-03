# Daily Planning Design

- Status: Proposed
- Depends on:
  - `go-web-app.md`
  - `routine.md`

## Context

The previous daily-plan skill gathered Google Calendar events, GitHub issues,
routine state, tasks, goals, notes, and training data; asked the user to choose
the day's priorities; generated summary and detail Markdown; posted both to
Discord; and stored a generic daily-plan entry. A cloud fallback generated a
plan autonomously when the user had not posted one.

The Go foundation intentionally retained only manual immutable plan revisions.
The next increment restores the smallest useful daily-planning loop inside the
Go web application:

1. gather Calendar, public GitHub issue, and routine context;
2. discuss priorities with the user through an AI chat;
3. review a structured proposal;
4. explicitly confirm an immutable plan revision.

## Goals

1. Keep the entire user experience in the Go web application.
2. Gather context deterministically in Go before invoking AI.
3. Provide a resumable, non-streaming chat UI.
4. Require an explicit preview and confirmation before storing a revision.
5. Store selected plan elements structurally while retaining Markdown summary
   and detail snapshots.
6. Preserve revision history for each local plan date.
7. Read public GitHub repositories without authentication for the MVP.
8. Read Google Calendar through user-authorized OAuth.
9. Enforce Zero Data Retention for all OpenRouter requests.
10. Leave the confirmed revision ready for a separately implemented daily run.

## Non-goals

- Personal task management. Free-form plan action items do not create a task
  system.
- Private GitHub repositories or GitHub mutations.
- Google Calendar writes.
- Discord publication or cloud fallback generation.
- Streaming model output.
- Autonomous finalization without user confirmation.
- Start-run execution, which has a separate draft design.
- Weekly or monthly planning workflows.

## Common plan envelope

The existing common tables remain:

```text
plans
plan_revisions
```

One logical plan is identified by:

```text
UNIQUE(user_id, kind, period_start)
```

For example, every daily revision for 2026-07-29 references the same `plans`
row with `kind = daily` and `period_start = 2026-07-29`.

Revision numbers are unique within the logical plan:

```text
UNIQUE(plan_id, revision)
```

The plan row is locked while allocating the next revision. The highest revision
number is current.

This envelope may also serve weekly or monthly plans, but it does not require
their structured content to match daily planning. Components are shared only
when their meaning and constraints remain the same. Future cadence-specific
designs may add their own extension tables.

## Structured revision content

The following optional component tables reference `plan_revisions.id`:

```text
plan_revision_github_issues
plan_revision_routines
plan_revision_action_items
plan_revision_calendar_events
```

Including `revision` in the relationship name is intentional: changing a
selected issue or routine creates a new revision without changing the content
of an older revision.

Every component row has a stable UUID and a `position` within its component
type. All `plan_revision_id` foreign keys are indexed.

### GitHub issues

```text
plan_revision_github_issues
id
plan_revision_id
repository_owner
repository_name
issue_number
title
html_url
position
```

The repository and issue number identify the external source. Title and URL are
snapshotted so plan history remains readable if the issue is later renamed or
deleted. This table does not mirror GitHub state and never becomes the source of
truth for whether the issue is open or complete.

### Routines

```text
plan_revision_routines
id
plan_revision_id
routine_id
position
```

`routine_id` references the routine domain. Routines referenced by a confirmed
plan cannot be deleted, while the revision Markdown remains the immutable
human-readable display snapshot. This avoids duplicating mutable routine names
and categories in structured rows.

### Action items

```text
plan_revision_action_items
id
plan_revision_id
title
note
position
```

Action items are plan-local free-form work. They are not tasks and have no
independent backlog or lifecycle. A future task design may add an optional task
reference without changing existing action items.

### Calendar events

Calendar events are constraints and context, not tasks.

```text
plan_revision_calendar_events
id
plan_revision_id
calendar_source_id
external_event_id
title
starts_at
ends_at
start_date
end_date
all_day
role
html_url
position
```

Timed events use `starts_at` and `ends_at`; all-day events use `start_date` and
`end_date`. A check constraint requires exactly one valid representation.

The revision stores all enabled events for the plan date and any later event
that the final plan explicitly mentions as relevant. It does not store the
entire 30-day input context.

Calendar events are presentation context, not the plan's prose. The Markdown
snapshots never narrate the day's timeline; the events surface only as a
minimal, low-footprint "time · title" list rendered from these structured rows.
The authoritative day view remains the user's calendar application.

`calendar_source_id` is nullable and uses `ON DELETE SET NULL`; the external
calendar and event IDs plus the display snapshot remain when a connection is
removed. Calendar source deletion therefore cannot damage confirmed plan
history.

### Notes and Markdown snapshots

`plan_revisions` gains optional `notes_markdown`. Existing
`summary_markdown` and `content_markdown` remain immutable rendered snapshots.
The structured rows support execution and querying; Markdown preserves the
human-readable result exactly as confirmed.

## Planning-session lifecycle

Planning sessions are working state, separate from confirmed revisions.

```text
daily_planning_sessions
id
user_id
plan_date
status
preview_json
confirmed_plan_revision_id
created_at
updated_at
finalized_at
```

Initial session states are:

- `gathering`;
- `context_blocked`;
- `chatting`;
- `reviewing`;
- `finalized`;
- `cancelled`.

Only one non-final session per user and plan date is allowed. The home page's
"Plan today" action resumes it instead of silently creating another.

Messages are stored while the session is active:

```text
daily_planning_messages
id
session_id
sequence
role
content
created_at
```

Normalized source context is stored as temporary JSONB working data:

```text
daily_planning_contexts
session_id
calendar_context
github_context
routine_context
calendar_status
github_status
routine_status
gathered_at
```

The JSON values use application-owned normalized schemas, never raw provider
responses. They exist so a process restart can resume the same conversation
without silently changing its evidence.

The user may explicitly refresh context. A refresh replaces the temporary
context, records the new gather time, invalidates any unconfirmed preview, and
adds a visible system note to the conversation.

## Gather failure flow

Sources are independent. If any source fails, the session moves to
`context_blocked` and displays each source status:

```text
Google Calendar: failed
GitHub: loaded
Routines: loaded
```

The user chooses one of:

- retry the failed source;
- continue without the failed source;
- cancel planning.

AI planning does not begin until the user chooses retry successfully or
explicitly continues. The eventual preview records which sources were
unavailable so confirmation is informed.

## Planning turns and confirmation

Each turn is a single, non-streaming structured request; there is no separate
review step. On every turn the model returns one object containing both a
conversational reply and its current draft of the plan.

1. The browser posts a user message.
2. The user message is saved.
3. The service sends the message history and normalized context to OpenRouter,
   requesting one structured object.
4. The model returns `assistant_message` plus its current draft: summary,
   content, and notes Markdown, and the selected GitHub issues, routines, action
   items, and calendar events.
5. `assistant_message` is saved as the assistant chat turn; the remaining fields
   become the session's current unconfirmed preview.
6. The handler returns the updated page: the conversation and the live plan
   draft beside it, with a Confirm action.

If the model request fails, the saved user message remains visible with a retry
action. Retrying is idempotent for that turn and does not duplicate messages.

The model does not call Calendar, GitHub, or routine tools. Go owns all external
I/O and passes provider-neutral context to AI.

Draft selections are sanitized on every turn: references absent from the session
context are dropped and duplicates removed, so an in-progress draft never blocks
on a stray hallucinated reference. The strict validation — required prose, length
and count limits, and every reference present in context — runs only at
confirmation.

When the draft is ready:

1. The user selects "Confirm plan."
2. The application re-validates the current preview strictly.
3. Confirmation creates the next immutable revision and all component rows in
   one transaction.

Revising an already confirmed day starts a new session seeded with the latest
revision and freshly gathered context. Confirmation creates the next revision;
it never updates the previous one.

After finalization, message content, temporary context, and preview JSON are
deleted. The session retains only status, timestamps, confirmed revision,
prompt version, selected model, and aggregate usage metadata.

## Presentation and rendering

The plan is authored as prose plus structured selections, and it must render as
a document a human can review, not as raw Markdown in a `<pre>` block.

### Markdown rendering

Summary, content, and notes Markdown, and assistant chat messages, are rendered
server-side to sanitized HTML. Rendering uses a CommonMark parser followed by an
allowlist sanitizer, so headings, lists, and wrapping display correctly and
nothing overflows its frame. AI output and user Markdown are untrusted and pass
through the same sanitizer; user chat messages are shown as escaped plain text
rather than parsed as Markdown, because user turns are instructions, not
authored documents.

### Structured selections over prose

`content_markdown` carries only prose: the day's priorities and the reasoning
and trade-offs behind them. Selected work does not appear as duplicated Markdown
lists. GitHub issues, routines, and action items render as their own linked,
structured lists beside the prose:

- during a session, the live preview joins the current draft's selections
  against saved session context to display titles and links;
- for a confirmed revision, the structured component rows are loaded back for
  display through a dedicated store read.

Snapshotted titles are preferred so history stays readable after an issue is
renamed or deleted.

On the home card, GitHub issues and action items render the same way: plain,
non-interactive list items. Routines are the one component type with a
completion model (`routine_events`), so the home card renders each selected
routine as either a one-click completion action (an optional note plus a
"Complete" button, defaulting to today's date) or, once completed for the
local day, a done state with no further action. GitHub issue completion means
the issue is closed on GitHub, which this application does not mutate; action
items have no completion model. Both stay display-only until a future design
gives them one — checking them would imply a persisted state the application
does not track.

### Prompt shaping

One prompt shapes every turn. It asks the model to return a single object whose
`assistant_message` carries the natural-language reply and whose remaining fields
carry the current plan draft. It instructs the model to keep selected work in the
structured fields, to avoid duplicating those selections in Markdown, and to never
narrate the calendar timeline in prose. The daily-planning prompt version is
advanced when this contract changes so confirmed revisions remain reproducible.

## Google Calendar integration

### Cloud and OAuth setup

The deployment uses a Google Cloud project with the Calendar API enabled and a
web-application OAuth client. The Go server requests offline access so it can
refresh access tokens when the user is not present.

The MVP requests the narrow read scopes:

- `https://www.googleapis.com/auth/calendar.events.readonly`;
- `https://www.googleapis.com/auth/calendar.calendarlist.readonly`.

Calendar writes are deferred. A later design may request
`calendar.events` incrementally and require a visible before/after approval
before every mutation.

Refresh tokens are stored encrypted, never logged, and never returned to the
browser after the OAuth callback. The encryption key is injected as a runtime
secret. Access tokens are refreshed on demand and are not persisted.

User OAuth grants are dynamic application credentials, so their encrypted
ciphertext is application data in PostgreSQL. The encryption key remains in the
deployment's secret manager; storing that key beside the ciphertext is
forbidden.

The MVP permits one Google Calendar connection per application user. Multiple
Google accounts and OpenID Connect subject storage are deferred to
[GitHub Issue #48](https://github.com/cloveclovedev/alt/issues/48).

### Calendar sources and interpretation

```text
google_calendar_connections
id
user_id
encrypted_refresh_token
granted_scopes
created_at
updated_at

calendar_sources
id
connection_id
external_calendar_id
display_name
enabled
role
planning_instructions
timezone
created_at
updated_at
```

`role` uses:

- `commitment`: a time constraint the plan should respect;
- `optional`: an opportunity or reminder that need not occupy the main
  schedule;
- `context`: information useful to planning but not itself a commitment.

The web application lists calendars discovered through Google and lets the user
enable them, assign a role, and edit interpretation instructions. Calendar IDs
and provider DTOs do not leave the Calendar adapter.

### Lookahead shaping

The service fetches enabled calendars for the local plan date through 30 days
after it. It does not pre-classify later events as "important."

The AI context is shaped by horizon:

- today: title, calendar, start/end, all-day state, location, and description;
- days 1-7: date/time, calendar, and title;
- days 8-30: date, calendar, and title.

Calendar role and planning instructions accompany every event. This keeps the
long horizon compact while allowing the model to identify preparation needs
without a hidden deterministic importance filter.

The plan does not invent exact time blocks for GitHub work. Existing Calendar
blocks such as a side-project period remain the scheduling constraint, while
the plan selects which issues to address.

## Public GitHub integration

Configured repositories are user data:

```text
github_repositories
id
user_id
owner
name
enabled
planning_instructions
created_at
updated_at
```

`(user_id, owner, name)` is unique.

The settings page validates that a configured repository is public and
reachable. At context-gather time, the Go adapter calls:

```text
GET /repos/{owner}/{repo}/issues?state=open&per_page=100
```

One response contains up to 100 issue or pull-request resources. The adapter
follows pagination when necessary and filters every item containing the
`pull_request` field. It retains issue number, title, labels, milestone,
assignees, update time, and URL in normalized session context.

Requests are unauthenticated in the MVP and therefore subject to GitHub's
IP-based unauthenticated limit. Context is gathered once per session rather
than on every chat turn.

Private repositories are deferred to a read-only GitHub App installed on the
user's personal account. The current adapter contract accepts repository
identity and normalized issue data so authentication can be added without
changing daily-planning domain types.

## Routine integration

The daily-planning service calls the routine service and receives active
routine statuses. It does not duplicate recurrence logic.

Candidates contain:

- stable routine ID;
- name and category;
- `overdue`, `today`, or `upcoming`;
- calculated due date when available;
- next available date for never-completed routines;
- persistent routine notes.

AI may recommend overdue, due-today, or strategically relevant upcoming
routines. Only routines explicitly confirmed by the user become
`plan_revision_routines` rows.

## AI provider and model selection

OpenRouter is the only inference adapter in the MVP.

The API key is an application secret injected at runtime. It is not stored in
PostgreSQL or accepted through the web application.

End-user AI bring-your-own-key support is deferred to
[GitHub Issue #47](https://github.com/cloveclovedev/alt/issues/47).

Model choices are purpose-based:

```text
ai_model_assignments
user_id
purpose
model_id
created_at
updated_at
```

The primary key is `(user_id, purpose)`. The MVP uses one purpose,
`daily_planning`, for both conversation and structured finalization.

Future purposes such as `weekly_planning` or
`daily_planning_finalization` can select different models without changing the
schema. A more specific future purpose may fall back to the broader assignment
until the user chooses an override.

The AI settings page obtains models from OpenRouter and shows only models that:

- have a Zero Data Retention endpoint;
- support text chat;
- support structured outputs.

ZDR is a mandatory application policy, not a user-disableable preference.
Every inference request includes provider preferences equivalent to:

```json
{
  "zdr": true,
  "data_collection": "deny",
  "require_parameters": true
}
```

If the selected model no longer satisfies the policy, planning stops and asks
the user to select another model. It never silently routes to a non-compliant
model.

OpenRouter prompt and response logging must remain disabled at the account
level as defense in depth.

## Usage metadata

Content is removed after finalization, but request metadata is retained for
cost and reliability diagnosis:

```text
ai_generations
id
session_id
purpose
model_id
provider_generation_id
prompt_version
prompt_tokens
completion_tokens
status
created_at
```

This table contains no prompt, response, Calendar title, or issue title.

## Web surfaces

Initial routes are:

```text
/                                      today entry card: status, start/resume, recent confirmed summary
/planning/daily/{date}                 conversation with the live plan draft, or the final plan
/planning/daily/{date}/context         gather or refresh context
/planning/daily/{date}/messages        post a message; returns the reply and updated draft
/planning/daily/{date}/confirm         confirm immutable revision
/plans/daily/{date}                    read-only view of a past confirmed plan
/settings/calendar                     Google connection and source rules
/settings/github                       public repository configuration
/settings/ai                           ZDR-compatible model selection
```

The home page is the single entry to today's planning. It is a compact card
showing today's status, a start-or-resume action that opens
`/planning/daily/{today}`, and a rendered summary of the most recent confirmed
plan. There is no separate "Plan today" navigation duplicating this action.

Free-form manual revision editing is removed. The AI planning flow is the only
path that creates a daily revision, so the home page no longer exposes a raw
Summary/content Markdown form.

Exact handler paths may change during implementation, but all transports call
the same application services.

## Security and privacy

- OAuth state and PKCE values are single-use and time-limited.
- Google refresh tokens are encrypted at rest.
- OpenRouter keys and token-encryption keys are runtime secrets.
- External provider responses and OAuth credentials are never logged.
- User-provided Markdown is rendered through a safe Markdown policy or escaped
  as plain text.
- AI output is untrusted input and receives the same validation and escaping as
  user input.
- Confirmation revalidates all IDs, ownership, lengths, enum values, and
  positions before writing.
- Source context and chat content are deleted after finalization.

## Error handling

- A source failure requires an explicit retry, continue, or cancel choice.
- A model timeout preserves the user message and offers retry.
- Invalid structured output is rejected and may be regenerated; it is never
  partially stored as a revision.
- A referenced item missing from session context blocks preview validation.
- A concurrent confirmation allocates one unique next revision under a plan-row
  lock; duplicate submission returns the already confirmed revision.
- A removed or no-longer-ZDR model blocks new AI requests until reconfigured.

## Testing

- Same local date resolves to one logical plan and successive revisions.
- Old revision components remain unchanged after a later revision.
- Session resume returns the same saved messages and context.
- Context refresh replaces working context and invalidates preview.
- Each source failure supports retry, explicit continuation, and cancel.
- Failed model calls do not duplicate user messages on retry.
- Preview rejects invented GitHub, Calendar, or routine identifiers.
- Confirmation writes revision and all component rows atomically.
- Finalization deletes content while retaining non-content usage metadata.
- Calendar context shaping uses the three documented horizons.
- GitHub pagination and pull-request filtering.
- Public repository validation and rate-limit errors.
- Every AI request enforces ZDR, denied data collection, and required
  parameters.
- A model that loses ZDR or structured-output capability is rejected.

## References

- Google Workspace project setup:
  <https://developers.google.com/workspace/guides/create-project>
- Google Calendar scopes:
  <https://developers.google.com/workspace/calendar/api/auth>
- Google OAuth for server-side web applications:
  <https://developers.google.com/identity/protocols/oauth2/web-server>
- Google Calendar event listing:
  <https://developers.google.com/workspace/calendar/api/v3/reference/events/list>
- GitHub repository issues:
  <https://docs.github.com/en/rest/issues/issues>
- GitHub REST API rate limits:
  <https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api>
- OpenRouter model listing:
  <https://openrouter.ai/docs/api/api-reference/models/get-models>
- OpenRouter structured outputs:
  <https://openrouter.ai/docs/guides/features/structured-outputs>
- OpenRouter ZDR enforcement:
  <https://openrouter.ai/docs/guides/features/zdr>
- OpenRouter provider routing:
  <https://openrouter.ai/docs/guides/routing/provider-selection>

## Decision log

This is a living design document. Each entry records a change to the design;
the sections above always describe the current intended design.

- 2026-07-29 — Initial daily-planning design: deterministic context gather,
  resumable non-streaming chat, structured proposal, and immutable confirmation.
  Implemented in [#43](https://github.com/cloveclovedev/alt/pull/43).
- 2026-08-02 — Added the presentation contract: single compact home entry card
  and removal of the manual revision form; Markdown rendered to sanitized HTML
  with escaped user chat; structured lists for selected issues, routines, and
  action items; calendar reduced to a minimal "time · title" reference never
  narrated in Markdown; and an advanced prompt version. Alternatives considered:
  a full inline session on the home page (rejected for a compact card plus
  navigation), dropping calendar events from confirmed plans entirely (rejected
  to keep constraints and history), and keeping the manual form as an advanced
  fallback (rejected to avoid two authoring paths). Tracked in
  [#61](https://github.com/cloveclovedev/alt/issues/61).
- 2026-08-03 — Unified the chat and proposal into a single structured turn.
  Every turn returns one object with a natural-language `assistant_message` plus
  the current plan draft (review-equivalent form and structured lists from the
  first turn), so the conversation and a live preview share one screen and the
  separate "Review plan"/"Back to chat" step is removed. Draft selections are
  sanitized (invalid references dropped) each turn; strict validation runs only
  at confirmation. Chosen over the earlier two-format flow (conversational chat,
  then a separate structured proposal) after that flow leaked proposal JSON into
  the chat and split the plan across two hard-to-review shapes. Trade-off: one
  structured-output call per turn. Tracked in
  [#61](https://github.com/cloveclovedev/alt/issues/61).
- 2026-08-03 — Made calendar events deterministic rather than a model
  selection. The model narrowed its calendar picks over turns, so the day's
  schedule appeared incomplete. Calendar events are the plan date's constraints:
  the draft and the confirmed revision now show and store every context event on
  the plan date, in start order, and the model no longer selects calendar events
  (that field left the proposal schema). Tracked in
  [#61](https://github.com/cloveclovedev/alt/issues/61).
- 2026-08-03 — Resolved the home card checkbox open question per component
  rather than uniformly: routines persist through the existing
  `routine_events` completion model (a one-click "Complete" action with an
  optional note, defaulting to today), while GitHub issues and action items
  stay display-only plain list items, since neither has a completion model
  the application owns. The routine completion action reuses
  `POST /routines/{routine_id}/complete`, which returns a small HTMX fragment
  instead of redirecting when called with `HX-Request`. Alternatives
  considered: checkboxes for all three component types with no persistence
  (rejected as misleading UI implying a state nothing tracks) and a new
  persisted completion table for action items (rejected as task-management
  scope creep, out of this design's non-goals). Tracked in
  [#64](https://github.com/cloveclovedev/alt/issues/64).
- 2026-08-03 — Refined prompt shaping so action items are reserved for
  free-form work not already captured by a selected GitHub issue, routine, or
  calendar event, since the model frequently produced an action item that only
  restated an already-selected issue (for example "Advance pepercheck #480"
  alongside `pepercheck#480` under GitHub issues), reading as duplicated
  content on the home card and confirmed-plan view. Advanced the prompt
  version accordingly. Tracked in
  [#66](https://github.com/cloveclovedev/alt/issues/66).
