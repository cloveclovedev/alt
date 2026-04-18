# X Post Scheduling & Approval Flow — Design Spec

Phase 2 of X post automation: webapp approval flow + scheduled posting via X API.

## Background

Phase 1 (#36) established automated draft generation (x-draft-cloud skill → entries DB + Discord notification). This spec covers Phase 2 (#40): reviewing/editing drafts on the webapp, scheduling, and posting via X API.

### Goals

- Review, edit, and approve X post drafts from webapp (`/posts`)
- Schedule posts to default time slots (12:00, 19:00 JST) with custom time override
- Post to X via API v2 (text + optional image)
- Attach images from PR screenshots or manual upload (Vercel Blob)

### Non-Goals

- Multi-account support (future PR)
- LLM-based re-editing from webapp
- Thread/reply support
- Analytics/engagement tracking

## Architecture Overview

```
x-draft-cloud (10:00/18:00 JST)
    → entries: type=x_draft, status=draft
    → Discord notification

webapp /posts
    → List drafts (draft/approved/posted/skipped)
    → Inline edit text, attach image (Vercel Blob)
    → Approve → auto-assign next available slot (12:00/19:00 JST)
    → Or specify custom scheduled_at
    → DB: status=approved, metadata.scheduled_at, metadata.image_url

cloud-scheduler (hourly)
    → x-post-cloud skill
    → Query: type=x_draft, status=approved, scheduled_at <= now
    → X API v2: upload image + post
    → DB: status=posted, metadata.tweet_id, metadata.posted_at
```

## Data Model

Uses existing `entries` table. No schema changes required.

```
entries (existing)
  type:     "x_draft"
  title:    Draft summary (one line)
  content:  Draft body text (≤280 chars)
  status:   "draft" | "approved" | "posted" | "skipped"
  tags:     ["peppercheck", "alt", ...]
  metadata: {
    "source_commits": ["repo:hash", ...],
    "source_memo_count": 3,
    "generated_at": "2026-04-10T10:00:00+09:00",
    "image_url": "https://....public.blob.vercel-storage.com/...",  # Vercel Blob URL
    "scheduled_at": "2026-04-10T12:00:00+09:00",
    "tweet_id": "1234567890",
    "posted_at": "2026-04-10T12:03:00+09:00"
  }
```

### Status Lifecycle

```
draft → approved (user approves on webapp, scheduled_at assigned)
draft → skipped  (user dismisses)
approved → posted (x-post-cloud posts via X API)
```

### Scheduling Logic (on approve)

1. Get all entries with `type=x_draft` and `status IN ('approved', 'posted')` to find occupied slots
2. Default slots: 12:00 and 19:00 JST daily
3. Find the next unoccupied slot after `now()`
4. Auto-assign as `scheduled_at`; user can override with date-time picker

Example: current time is 4/10 10:30
- 4/10 12:00 available → assign 4/10 12:00
- 4/10 12:00 occupied → check 4/10 19:00
- 4/10 19:00 occupied → check 4/11 12:00
- Custom time specified → use that instead

## Components

### 1. Webapp — `/posts` page

**Route:** `webapp/src/app/posts/page.tsx`

**Layout:**
- New "Posts" tab in nav bar (after Body)
- Grouped by status: Draft (pending review) → Approved (scheduled) → History (posted/skipped)

**Draft card (status=draft):**
- Inline editable textarea with real-time character count (/280)
- Image attachment: file picker → upload to Vercel Blob → preview thumbnail
- PR image (if `metadata.image_url` already set from draft generation): displayed as preview, replaceable
- "Approve" button: assigns next available slot, shows scheduled time, allows custom time via date-time picker
- "Skip" button: sets status=skipped

**Approved card (status=approved):**
- Read-only text preview + image preview
- Scheduled time display
- "Cancel" button: reverts to draft status, clears scheduled_at (image_url is preserved)

**History section (status=posted/skipped):**
- Collapsed by default, expandable
- Shows posted_at, tweet_id (link to post) for posted items

**Server Actions:**
- `approveDraft(id, scheduledAt, imageUrl?)` → update status=approved, set metadata
- `skipDraft(id)` → update status=skipped
- `cancelApproval(id)` → revert status=draft, clear scheduled_at
- `updateDraftContent(id, content)` → update content field
- `uploadImage(formData)` → upload to Vercel Blob, return URL

**Queries (add to queries.ts):**
- `getXDrafts()` → entries where type=x_draft, ordered by status priority then created_at
- `getOccupiedSlots()` → scheduled_at values of approved/posted drafts for slot calculation

### 2. Vercel Blob — Image Storage

**Setup:**
- Add `@vercel/blob` to webapp dependencies
- Create Blob store in Vercel dashboard (Hobby plan, free tier: 5GB storage)

**Upload flow:**
1. User selects image on `/posts` page
2. Client upload via `@vercel/blob` (avoids server-side bandwidth)
3. Returns public URL
4. URL saved to `metadata.image_url` on approve

**Image from PR (draft generation):**
- x-draft-cloud skill extracts image URLs from PR body markdown (`![...](url)`)
- GitHub-hosted images (user-attachments) are publicly accessible
- URL saved to `metadata.image_url` at draft creation time

### 3. x-post-cloud skill (new)

**Path:** `.claude/skills/x-post-cloud/SKILL.md`

Cloud-invoked skill, called by cloud-scheduler every hour.

**Execution flow:**
1. Install deps: `uv sync`
2. Query approved drafts due for posting:
   ```bash
   uv run alt-db --json entry list --type x_draft --status approved
   ```
   Filter: `metadata.scheduled_at <= now` (in JST)
3. For each draft:
   a. If `metadata.image_url` exists:
      - Download image to temp file
      - Upload to X: `POST https://api.x.com/2/media/upload` → get `media_id`
   b. Post to X: `POST https://api.x.com/2/posts` with text content (+ `media_id` if image)
   c. Update entry:
      ```bash
      uv run alt-db entry update <id> --status posted \
        --metadata '{"tweet_id": "...", "posted_at": "..."}'
      ```
   d. Post confirmation to Discord:
      ```bash
      uv run alt-discord post <channel_id> "Posted to X: <tweet_url>"
      ```
4. If no drafts are due, log and exit immediately.

**X API calls:** Made directly via `curl` with OAuth headers (skill-orchestrated, no Python wrapper needed).

**Error handling:**
- If X API returns error: log to Discord, keep status=approved (retry next hour)
- If image upload fails: post text-only, note in Discord

### 4. Cloud Scheduler Changes

**Cron:** Change from `0 1,6,9,12,15,21 * * *` to `0 * * * *` (hourly)

**Dispatch table update:**

x-post-cloud runs **every hour** as the first skill. All other skills keep their existing schedule unchanged:

| Hour | Day     | Additional skills (after x-post-cloud) |
|------|---------|----------------------------------------|
| 0    | Daily   | nutrition-check-cloud |
| 6    | Daily   | nutrition-check-cloud |
| 10   | Sun (7) | weekly-plan-cloud → daily-plan-cloud → x-draft-cloud → nutrition-check-cloud |
| 10   | 1-6     | daily-plan-cloud → x-draft-cloud → nutrition-check-cloud |
| 15   | Daily   | nutrition-check-cloud |
| 18   | Daily   | x-draft-cloud |
| 21   | Daily   | nutrition-check-cloud |
| *    | Daily   | (x-post-cloud only) |

### 5. alt.toml Changes

```toml
[x]
default_post_times = ["12:00", "19:00"]  # renamed from post_times, used for auto-scheduling
```

## Configuration & Environment Variables

### Webapp (Vercel)
- `BLOB_READ_WRITE_TOKEN` — Vercel Blob access (auto-set by Vercel when Blob store is created)

### Cloud Environment (existing + new)
- `DISCORD_BOT_TOKEN` — existing
- `DATABASE_URL` — existing
- `GH_TOKEN` — existing
- `X_CONSUMER_KEY` — X API OAuth 1.0a consumer key (new)
- `X_CONSUMER_SECRET` — X API OAuth 1.0a consumer secret (new)
- `X_ACCESS_TOKEN` — X API OAuth 1.0a access token (new)
- `X_ACCESS_TOKEN_SECRET` — X API OAuth 1.0a access token secret (new)

Note: X API v2 posts endpoint supports both OAuth 1.0a and OAuth 2.0 User Context. OAuth 1.0a is simpler for a single-user bot scenario (no token refresh needed).

## Prerequisites (User Action Required)

1. **X API setup** — Register at developer.x.com, set up pay-per-use billing (~$10 initial charge), create app with Read+Write permissions, generate OAuth 1.0a keys
2. **Vercel Blob** — Create Blob store in Vercel dashboard for the webapp project
3. **Cloud environment** — Add X API credentials as environment variables

## Testing

- Webapp: manual verification on Vercel preview deploy
- x-post-cloud: manual "Run now" on cloud trigger after approving a draft
- End-to-end: generate draft → approve on webapp → verify post appears on X at scheduled time
