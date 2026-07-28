# Routine Management Design

- Status: Proposed
- Date: 2026-07-29
- Depends on: `2026-07-27-go-web-app-design.md`

## Context

The previous implementation stored routine definitions in a generic JSON
configuration object and completion history in generic entries. Daily and
weekly planning then reconstructed routine state by matching a completion
entry's title to a definition's title.

That model supported early experimentation, but title-based identity made
renames unsafe and the generic storage left validation to agent instructions.
The Go application needs a typed routine domain that can be edited through the
web application and consumed deterministically by daily planning.

## Goals

1. Let a user create, edit, activate, deactivate, and inspect routines in the
   web application.
2. Give each routine stable identity independent of its display name.
3. Record completion history.
4. Derive one current due date and one of three states: `overdue`, `today`, or
   `upcoming`.
5. Support interval-based recurrence constrained by available weekdays and
   active months.
6. Expose a transport-neutral routine status view to daily planning.
7. Let users manage and order routine categories.

## Non-goals

- Cron expressions or calendar-style recurrence rules.
- Fixed weekday recurrences such as "every Monday and Wednesday" independent
  of completion history.
- Notifications or scheduled background jobs.
- Automatically completing a routine from an external service.
- Daily-run sequencing, which is designed separately.

## Domain model

### Routine categories

Categories are user-managed data, not a PostgreSQL enum. A user can add,
rename, deactivate, and reorder them without a schema migration.

```text
routine_categories
id
user_id
name
position
active
created_at
updated_at
```

`(user_id, name)` is unique. Category names are compared after trimming but
retain user-selected casing for display. `position` controls display order;
ties fall back to `name`, then `id`, so concurrent edits remain deterministic.

A category referenced by a routine cannot be deleted. It can be deactivated,
and a category with no routines may be deleted through an explicit action.

### Routines

```text
routines
id
user_id
category_id
name
notes
status
interval_days
available_weekdays
active_months
created_at
updated_at
```

`status` uses a `routine_status` enum with `active` and `inactive`.

`interval_days` is a positive integer. It is the number of local calendar days
between an actual completion date and the unconstrained next due date.

`available_weekdays` is an array of ISO weekday numbers:

```text
1 = Monday
2 = Tuesday
...
7 = Sunday
```

An empty array means any weekday. The database constrains every member to
`1..7`; the service removes duplicates and stores members in ascending order.
The web UI labels the field "Days when this can be done."

`active_months` is an array of month numbers `1..12`. An empty array means every
month. The service likewise removes duplicates and stores members in ascending
order.

`notes` contains persistent instructions about the routine. Notes about one
particular completion belong to the event instead.

`(user_id, name)` is unique among routines. Renaming a routine does not affect
history because events reference `routines.id`.

### Routine events

```text
routine_events
id
routine_id
completed_on
note
created_at
updated_at
```

`completed_on` is a `date`, not an instant. Recurrence is defined in the user's
local calendar and does not need time-of-day precision. `created_at` still uses
`timestamptz` for audit ordering.

The latest event by `(completed_on, created_at, id)` is the recurrence anchor.
Multiple events on the same day are allowed so an incorrect event can be
corrected without imposing an unrelated uniqueness rule.

All foreign-key columns have supporting indexes. Deletes cascade from a
routine to its events only when the routine itself is eligible for permanent
deletion.

## Due-date calculation

Due dates are derived in Go and are not persisted. Persisting them would allow
schedule changes or corrected history to leave stale due dates behind.

For a routine with an event:

```text
base due date = latest completed_on + interval_days
due date      = first date on or after the base due date that:
                - falls in active_months, when restricted; and
                - falls on available_weekdays, when restricted
```

Restrictions only move the due date later. They never move it earlier.

Example:

```text
latest completed_on: 2026-07-20
interval_days:       8
base due date:       2026-07-28 (Tuesday)
available_weekdays:  [6, 7]
due date:            2026-08-01 (Saturday)
```

The state compares the derived due date with the user's current local date:

```text
due date < today  -> overdue
due date = today  -> today
due date > today  -> upcoming
```

There is no `due soon` state.

An active routine with no events is immediately `overdue`, because the missing
last-completion date is treated as evidence that it has not been performed
recently. Its `due_on` is absent, and the status view separately exposes
`next_available_on` so the UI can show when it can next be performed.

Inactive routines are excluded from due calculation and planning candidates.
Their history remains visible.

A routine may be deleted only when it has no events and is not referenced by a
confirmed plan revision. Otherwise it must be deactivated. This keeps foreign
keys intact and prevents historical plans from losing their routine identity.

## Creation flow

The creation form requires:

- name;
- category;
- interval in days;
- available weekdays, optional;
- active months, optional;
- notes, optional;
- last completed date.

The last completed date is required by default. Creating the routine and its
initial event occurs in one transaction.

The user may explicitly select "Never completed or unknown." That creates the
routine without an event and makes it immediately overdue.

## Completion and correction

Completing a routine creates an event with the user's local date and an optional
note. The next due date is calculated from that actual completion date, not from
the previous scheduled due date.

The history screen supports correcting the completion date or note and deleting
an event recorded by mistake. These are explicit correction operations and do
not change the routine definition.

Deleting the last event can make the routine immediately overdue. The UI shows
that consequence before confirmation.

## Web experience

The routine area includes:

- a category manager with add, rename, activate/deactivate, and reorder actions;
- an active-routine list grouped by category and ordered by category position;
- status sections for `overdue`, `today`, and `upcoming`;
- a routine form for definition and initial completion data;
- a routine detail page with definition, derived status, and event history;
- completion and correction actions.

The list hides inactive routines by default but offers an inactive filter.

## Application boundary

The routine service exposes a provider-neutral status model:

```go
type Status struct {
    RoutineID      string
    Name           string
    CategoryID     string
    CategoryName   string
    State          DueState
    DueOn          *time.Time
    NextAvailableOn *time.Time
    Notes          string
}
```

Daily planning consumes this model rather than reimplementing recurrence
calculation.

## Validation and constraints

- `interval_days > 0`;
- every available weekday is in `1..7`;
- every active month is in `1..12`;
- names are non-blank after trimming;
- a routine and category belong to the same user;
- event dates cannot be later than the user's current local date;
- category and routine foreign keys are indexed.

The service validates inputs for useful user-facing errors, while PostgreSQL
constraints protect stored data independently.

## Testing

- Due date without weekday or month restrictions.
- Due date shifted forward to the next available weekday.
- Due date shifted across an inactive month.
- Combined weekday and month restrictions.
- Leap years, month ends, and year boundaries.
- `overdue`, `today`, and `upcoming` comparisons in the user's timezone.
- Missing history is immediately overdue and has a next available date.
- Completing a routine anchors the next interval to the actual completion date.
- Inactive routines are absent from planning candidates.
- Rename preserves event history.
- Category ownership and delete restrictions.
- Creation writes the routine and its initial event atomically.

## Deferred extensions

- Additional recurrence strategies can be introduced as a new schedule type
  only after a concrete routine cannot be expressed by this model.
- Daily-run completion will reuse the same completion service.
- Weekly and monthly planning may consume the same status model without forcing
  their plan structures to match daily planning.
