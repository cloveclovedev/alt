"""Home Assistant REST API client."""

import json
import os
import urllib.request
import urllib.error


class HomeAssistantClient:
    """Thin wrapper around the Home Assistant REST API."""

    def __init__(self, url: str, token: str):
        self.url = url.rstrip("/")
        self.token = token

    @classmethod
    def from_env(cls) -> "HomeAssistantClient":
        """Create client from HA_URL and HA_TOKEN environment variables."""
        url = os.environ.get("HA_URL", "")
        token = os.environ.get("HA_TOKEN", "")
        missing = []
        if not url:
            missing.append("HA_URL")
        if not token:
            missing.append("HA_TOKEN")
        if missing:
            raise RuntimeError(f"Missing environment variables: {', '.join(missing)}")
        return cls(url, token)

    def _request(self, method: str, path: str, data: dict | None = None) -> dict | list:
        """Make an authenticated request to the HA REST API."""
        body = json.dumps(data).encode() if data else None
        req = urllib.request.Request(
            f"{self.url}{path}",
            data=body,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Content-Type": "application/json",
            },
            method=method,
        )
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read())
        except urllib.error.HTTPError as e:
            error_body = e.read().decode()
            raise RuntimeError(f"HA API error ({e.code}): {error_body}") from e
        except urllib.error.URLError as e:
            raise RuntimeError(f"HA connection error: {e.reason}") from e

    def get_state(self, entity_id: str) -> dict:
        """Get the current state of an entity."""
        return self._request("GET", f"/api/states/{entity_id}")

    def call_service(self, domain: str, service: str, data: dict) -> list:
        """Call a Home Assistant service."""
        return self._request("POST", f"/api/services/{domain}/{service}", data)

    def tts(self, message: str, entity_id: str) -> list:
        """Send a TTS message to a media player entity."""
        return self.call_service("tts", "speak", {
            "entity_id": entity_id,
            "message": message,
        })
