# Bitwarden Secrets Manager Development Injection

## Scope

This repository uses the shared `development` Bitwarden Secrets Manager
project. The local machine account can read that project only. It must never
receive staging or production access tokens.

Create the following application-prefixed secrets in the `development`
project:

```text
ALT_OPENROUTER_API_KEY
ALT_GOOGLE_OAUTH_CLIENT_ID
ALT_GOOGLE_OAUTH_CLIENT_SECRET
ALT_CALENDAR_TOKEN_ENCRYPTION_KEY
```

The Calendar redirect URL is non-secret development configuration. Compose sets
it to:

```text
http://localhost:28080/settings/calendar/callback
```

`ALT_CALENDAR_TOKEN_ENCRYPTION_KEY` must be a Base64 encoding of 32 random
bytes. Generate it once with `openssl rand -base64 32` and store only the
result in Bitwarden.

## Local setup

1. Install the native `bws` executable from the official Bitwarden Secrets
   Manager CLI release and place it on `PATH`.
2. Create a read-only `local-development` machine account and grant it access
   to the `development` project.
3. Store its access token in macOS Keychain with service name
   `bws-local-access-token` and your macOS account name. Do not put it in
   `.env`, shell startup files, Docker Compose files, or the repository.
4. Run:

   ```bash
   make dev-bws PROJECT_ID=<development-project-id>
   ```

The helper reads the token from Keychain into its process tree, then invokes
`bws run --project-id` for the trusted Compose command. Only the selected
project's `ALT_*` secret values reach the `web` container; the BWS access token
is not passed to it.

The application gives `ALT_*` values precedence over generic names. Generic
names remain supported for isolated deployment environments where the secret
scope already belongs to alt alone.
