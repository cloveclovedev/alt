# Vercel Deploy Design — alt Dashboard

## Summary

Deploy the Next.js dashboard (`webapp/`) to Vercel with GitHub OAuth authentication. No code changes required — configuration only via Vercel Dashboard and GitHub.

## Architecture

```
Browser → Vercel (*.vercel.app)
            ├── Next.js (webapp/)
            ├── NextAuth (GitHub OAuth - production App)
            └── @neondatabase/serverless (HTTP) → Neon Postgres
```

- **Hosting**: Vercel Hobby plan
- **Framework**: Next.js (auto-detected by Vercel)
- **Root Directory**: `webapp/`
- **Database**: Neon Postgres via `@neondatabase/serverless` HTTP driver (no connection pooling needed)
- **Auth**: NextAuth v5 with GitHub OAuth provider, single-user restriction via `ALLOWED_GITHUB_USER_ID`

## Environment Separation

| Environment | OAuth App | AUTH_SECRET | Config Location |
|---|---|---|---|
| Local dev | Existing dev OAuth App | Existing value | `webapp/.env.local` |
| Production | New `alt-webapp-prod` OAuth App | New value via `openssl rand -hex 32` | Vercel Dashboard |

`DATABASE_URL` and `ALLOWED_GITHUB_USER_ID` are shared across both environments.

## Vercel Project Settings

- **Plan**: Hobby (free)
- **GitHub repo**: Connected, auto-deploy on push to `main`
- **Root Directory**: `webapp/`
- **Build/Output**: Defaults (Next.js auto-detected)

## Environment Variables (Vercel Dashboard)

| Variable | Source | Notes |
|---|---|---|
| `DATABASE_URL` | Neon Dashboard | Same connection string as local |
| `AUTH_SECRET` | `openssl rand -hex 32` | Separate from local dev |
| `AUTH_GITHUB_ID` | New production OAuth App | |
| `AUTH_GITHUB_SECRET` | New production OAuth App | |
| `ALLOWED_GITHUB_USER_ID` | Same as local | Makoto's GitHub user ID |

## GitHub OAuth App (Production)

- **Application name**: `alt-webapp-prod`
- **Homepage URL**: `https://alt-mauve.vercel.app`
- **Authorization callback URL**: `https://alt-mauve.vercel.app/api/auth/callback/github`

## Deployment Steps

1. Create Vercel project — import GitHub repo, set Root Directory to `webapp/`
2. Create production GitHub OAuth App with Vercel domain callback URL
3. Set environment variables in Vercel Dashboard
4. Trigger deploy (or push to main)
5. Verify deployment

## Verification Checklist

- [x] Vercel build succeeds
- [x] `/login` page renders
- [x] GitHub OAuth login completes successfully
- [x] Dashboard pages render: `/`, `/entries`, `/routines`
- [x] Unauthenticated access redirects to `/login`
