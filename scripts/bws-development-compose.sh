#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <Bitwarden development project ID>" >&2
  exit 2
fi

if ! command -v bws >/dev/null 2>&1; then
  echo "bws is not installed; install the Bitwarden Secrets Manager CLI first" >&2
  exit 127
fi

if [ -z "${BWS_ACCESS_TOKEN:-}" ] && command -v security >/dev/null 2>&1; then
  # The user creates this Keychain item once with the shared development
  # machine-account token. The value is read only into this process tree.
  BWS_ACCESS_TOKEN="$(security find-generic-password -a "$(id -un)" -s "bws-local-access-token" -w 2>/dev/null || true)"
  export BWS_ACCESS_TOKEN
fi

if [ -z "${BWS_ACCESS_TOKEN:-}" ]; then
  echo "BWS_ACCESS_TOKEN is required; add bws-local-access-token to macOS Keychain or set it for this one command" >&2
  exit 2
fi

# The project ID is not secret. bws injects only this project's ALT_* values
# into Docker Compose; BWS_ACCESS_TOKEN itself is not listed in compose.yaml.
exec bws run --project-id "$1" -- docker compose --env-file /dev/null up --build
