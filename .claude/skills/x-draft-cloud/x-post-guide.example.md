# X Post Tone Guide (Example)

Copy this file to `x-post-guide.md` (gitignored) and adapt it to your own voice.
The `x-draft-cloud` skill reads `x-post-guide.md` if present.

## Tone

- Calm and casual — the way an engineer normally talks
- Short sentences. Don't pad short content.
- No excessive emojis, no `!!!`, no marketing language.
- Translate technical activity into user-facing impact (don't expose PR/issue numbers).

## Good examples

- "Added a notifications toggle to MyApp. You can switch it on/off from settings."
- "Fixed an auth bug — sessions no longer drop right after login."
- "Spent the day on UI polish. Small changes, but the app feels noticeably nicer to use."

## Bad examples

- "🎉✨ NEW FEATURE LAUNCHED!! Notifications are HERE!! Try it now!! 🚀" (over-the-top)
- "Today I merged PR #42 and closed 3 issues" (too technical)
- "Worked hard today, will work hard tomorrow!" (no content)
- "Please use my product MyApp!" (advertising tone)

## Anti-patterns

- Heavy emoji use
- Strings of `!`
- Calling out to the audience ("everyone", "please")
- Surfacing PR/issue numbers as-is
- "Released!" with no specifics
- Hashtag spam
- Stretching short content to look longer

## Post types

| Type | Focus | Example |
|---|---|---|
| progress | What changed + user impact | "Added a notifications toggle to MyApp." |
| technical | What was chosen + why | "Picked Riverpod for state management — fewer Provider headaches." |
| problem-solution | What broke + how it was fixed | "Hit a Neon connection pool issue. Turned out to be pgbouncer's prepared statement setting." |
| reflection | An observation, kept short | "Working in OSS gets you in the habit of writing down design intent." |

## Algorithm-friendly rules

- No links in the main post. Put links in a self-reply.
- At most 2 hashtags. Prefer specific tech names (`#Flutter`, `#Riverpod`). For reflections, `#OSS` or similar works.
- Don't post internal implementation details from private repos. Focus on the "why" and the design decision.
- When posting from a design doc, focus on the "why" of a tech choice — skip the "how".
- Attach images when available (PR screenshots, UI captures).
