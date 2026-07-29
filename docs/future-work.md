# Future Work

This index tracks accepted work that is intentionally outside the current MVP.
Detailed design records remain in `docs/designs/`.

## Next

### Firebase Authentication and request-scoped users

Replace the fixed development `ALT_USER_ID` with a user resolved from a
verified Firebase ID token. Map each Firebase UID to an application `users.id`,
then pass that user identity through the HTTP boundary into feature services.

This should be completed before expanding the product beyond the current
single-user MVP. Firebase login credentials must remain separate from the
Google Calendar OAuth connection used to access Calendar data.

## Later

### User-provided AI keys

Support encrypted, user-owned OpenRouter keys rather than a service-owned key
when the application becomes multi-user. See [AI BYOK future work](designs/2026-07-29-ai-byok-future-work.md).

### Multiple Google Calendar accounts

Allow more than one connected Google account for one application user, with
Google subject identity and connection selection. See [Calendar connection future work](designs/2026-07-29-calendar-connection-future-work.md).

### Planning UI design pass

Replace the functional MVP views with a deliberate visual system, including
Markdown rendering, session history hierarchy, settings layout, and responsive
interaction feedback.
