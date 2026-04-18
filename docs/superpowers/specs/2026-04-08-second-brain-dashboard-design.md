# Second Brain Dashboard — Design

## Problem

The Second Brain knowledge store (Neon Postgres) has a CLI for data access, but no visual overview. Browsing goals, memos, and routine status requires running terminal commands. A web dashboard provides at-a-glance visibility into personal knowledge and routines.

## Solution

A Next.js (App Router) web dashboard deployed on Vercel. Read-only initial scope with authentication. Connects directly to Neon Postgres via the serverless driver.

## Architecture

```
Browser → Vercel (Next.js App Router)
                 ↓
           Auth.js middleware (GitHub OAuth, user ID whitelist)
                 ↓
           Server Components → @neondatabase/serverless → Neon Postgres
```

- **Server Components** query the DB directly. No API routes needed for read-only.
- All pages require authentication. Unauthenticated requests redirect to `/login`.
- Neon Serverless Driver uses HTTP over WebSocket — works on Vercel Edge and Serverless.
- When write/update features are needed later, add Server Actions or Route Handlers. The DB connection layer supports read and write.

## Repository Structure

Monorepo — `webapp/` directory alongside existing Python CLI and DB schema.

```
alt/
├── db/                    # Atlas HCL schema (existing)
├── src/alt_db/            # Python CLI (existing)
├── webapp/                # Next.js dashboard (new)
│   ├── package.json
│   ├── next.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── components.json    # shadcn/ui config
│   ├── .env.local         # Local env vars (gitignored)
│   └── src/
│       ├── app/
│       │   ├── layout.tsx          # Root layout
│       │   ├── page.tsx            # Dashboard (/)
│       │   ├── entries/
│       │   │   └── page.tsx        # Entry list
│       │   ├── routines/
│       │   │   └── page.tsx        # Routine status
│       │   └── login/
│       │       └── page.tsx        # Login page
│       ├── lib/
│       │   ├── db.ts               # Neon connection helper
│       │   ├── queries.ts          # DB query functions
│       │   └── auth.ts             # Auth.js config
│       ├── components/
│       │   └── ui/                 # shadcn/ui components
│       └── middleware.ts           # Auth guard
```

Vercel deployment targets `webapp/` as root directory.

## Authentication

**Auth.js v5 + GitHub OAuth Provider.** Single-user personal project — only one GitHub account is permitted.

### Flow

1. Unauthenticated user visits any page
2. `middleware.ts` detects no session → redirect to `/login`
3. User clicks "Sign in with GitHub" → GitHub OAuth flow
4. Auth.js callback receives GitHub profile
5. `callbacks.signIn` checks `profile.id` against `ALLOWED_GITHUB_USER_ID` env var
6. Match → session created (JWT). No match → sign-in rejected.

### Why GitHub User ID (not email or username)

- GitHub user ID is an immutable numeric identifier
- Email and username can be changed, so they are unreliable for access control
- The numeric ID is returned in the OAuth profile and cannot be spoofed

### Session

JWT strategy (no DB table needed). Auth.js default `maxAge` (30 days). Auto-logout is intentionally omitted — personal devices only, and GitHub re-login is trivial, so idle timeout adds friction without meaningful security benefit.

### Environment Variables

```
AUTH_SECRET=<random-string>           # Session encryption
AUTH_GITHUB_ID=<oauth-app-client-id>
AUTH_GITHUB_SECRET=<oauth-app-secret>
ALLOWED_GITHUB_USER_ID=<numeric-id>
```

## Data Access

### Connection (`src/lib/db.ts`)

Thin wrapper around `@neondatabase/serverless`. Reads `DATABASE_URL` from server environment.

```ts
import { neon } from "@neondatabase/serverless";

export const sql = neon(process.env.DATABASE_URL!);
```

### Queries (`src/lib/queries.ts`)

Mirrors the Python CLI's query layer. All functions run server-side only.

- `getActiveGoals()` — entries where type=goal, status=active
- `getRecentMemos(days: number)` — entries where type=memo, created within N days
- `getUpcomingDeadlines(days: number)` — goals with target_date in metadata within N days
- `listEntries(filters)` — entries filtered by type, status, tag, search query
- `getLatestRoutineEvents()` — most recent event per routine (DISTINCT ON)

## Pages

### `/` — Dashboard

Overview page with three card sections:

- **Active Goals** — List of goals with status=active. Goals with approaching deadlines (within 7 days, from `metadata.target_date`) highlighted.
- **Recent Memos** — Memos from the last 7 days. Title and truncated content.
- **Deadline Alerts** — Goals due within 7 days, prominently displayed if any exist.

### `/entries` — Entry List

- Filter controls: type (dropdown), status (dropdown), tag (text input)
- Text search: searches title + content (ILIKE)
- Filter state managed via URL search params (bookmarkable, shareable)
- Results displayed as a table or card list
- Tags rendered as badges
- Pagination if entry count grows

### `/routines` — Routine Status

- Displays the latest event per routine from `routine_events`
- Shows: routine name, category, last completed date, optional note
- Overdue calculation is **deferred** until routine definitions are migrated from YAML to DB (see mkuri/alt#30). Initial version shows last completion dates only.

## UI

- **Tailwind CSS** for styling
- **shadcn/ui** for component library (Button, Card, Table, Badge, Input, Select, etc.)
- Design guidance from ui-ux-pro-max skill for layout, color, and typography decisions
- Dark mode support via Tailwind's `dark:` classes and shadcn/ui theming
- Responsive layout (usable on mobile, optimized for desktop)

## Security

| Layer | Measure |
|---|---|
| Auth guard | `middleware.ts` protects all routes. Only `/login` and `/api/auth/*` are public |
| User restriction | GitHub OAuth `profile.id` (immutable numeric ID) checked against allowlist |
| Server-side data | All DB queries in Server Components. No sensitive data in client JS bundles |
| DB connection | `DATABASE_URL` is server-only env var (no `NEXT_PUBLIC_` prefix). Neon SSL required |
| HTTPS | Vercel default. Local dev: DB credentials stay server-side, never sent to browser |
| Security headers | `next.config.ts` sets `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin` |
| OAuth app | GitHub OAuth App scoped to read-only user profile. No repo or org access requested |

## Deployment

- **Platform:** Vercel (Hobby plan, free)
- **Root Directory:** `webapp/`
- **Build Command:** `npm run build` (Next.js default)
- **Environment Variables:** Set via Vercel Dashboard:
  - `DATABASE_URL`
  - `AUTH_SECRET`
  - `AUTH_GITHUB_ID` / `AUTH_GITHUB_SECRET`
  - `ALLOWED_GITHUB_USER_ID`

## Out of Scope

- Write/update operations from dashboard (add via Server Actions when needed)
- Routine overdue calculation (blocked on mkuri/alt#30 — routine definitions DB migration)
- pgvector semantic search
- Mobile app
