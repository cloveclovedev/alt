# Calendar Connection Future Work

- Status: Proposed
- Date: 2026-07-29
- Depends on: `2026-07-29-daily-planning-design.md`

## MVP decision

The MVP supports exactly one Google Calendar connection for each application
user. The application user ID is the connection owner, so the connection table
does not store a Google OpenID Connect subject.

This keeps the Calendar authorization request limited to the two Calendar
read-only scopes required for daily planning:

- `https://www.googleapis.com/auth/calendar.events.readonly`;
- `https://www.googleapis.com/auth/calendar.calendarlist.readonly`.

## When to revisit

Add multiple Google account connections only when a user needs to combine
separate accounts. That work must request the `openid` scope, validate the ID
token, and store its stable `sub` claim with each connection. The connection
identity must then be unique per application user and Google subject.

Do not infer this identity from a Calendar ID or an email-like display value.
Calendar IDs are provider resource identifiers, not a stable OAuth account
identity.
