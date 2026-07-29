# AI Bring-Your-Own-Key Future Work

- Status: Proposed
- Date: 2026-07-29
- Depends on: `2026-07-29-daily-planning-design.md`

## Decision trigger

The MVP uses an operator-owned OpenRouter API key injected as a runtime secret.
Do not make the operator financially responsible for arbitrary end-user model
usage when alt becomes a multi-user service.

At that point, add an explicit bring-your-own-key (BYOK) design before enabling
AI planning for additional users.

## Required design properties

- Require an authenticated application user before accepting a key.
- Store a user's provider key only as ciphertext in PostgreSQL. Keep the
  envelope-encryption key in the deployment secret manager, never beside the
  ciphertext.
- Do not return a stored key to the browser, include it in logs, or retain it
  in AI-generation metadata.
- Provide explicit replace, test, and delete actions. Delete must remove the
  ciphertext immediately.
- Use the key only for the owning user's inference request and enforce product
  authorization at that boundary.
- Document provider billing, rate-limit, and revocation behavior clearly.

Bitwarden Secrets Manager remains the system of record for application and
deployment secrets. It is not a store for individual users' BYOK credentials.

## Alternative

An ephemeral BYOK flow can avoid database storage, but it cannot reliably
resume a planning session after a browser or process restart. Reconsider it
only if persistent user keys are unacceptable and that usability trade-off is
intentional.
