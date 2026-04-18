"""Read messages from a Discord channel, including threads."""

import json
import os
import urllib.request
import urllib.parse
from datetime import datetime

DISCORD_API = "https://discord.com/api/v10"


def timestamp_to_snowflake(iso_timestamp: str) -> str:
    """Convert ISO 8601 timestamp to Discord snowflake ID."""
    dt = datetime.fromisoformat(iso_timestamp)
    unix_ms = int(dt.timestamp() * 1000)
    discord_epoch = 1420070400000
    return str((unix_ms - discord_epoch) << 22)


def _discord_get(path: str) -> list | dict:
    """Make authenticated GET request to Discord API."""
    token = os.environ["DISCORD_BOT_TOKEN"]
    req = urllib.request.Request(
        f"{DISCORD_API}{path}",
        headers={
            "Authorization": f"Bot {token}",
            "User-Agent": "DiscordBot (https://github.com/mkuri/alt, 0.1.0)",
        },
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode("utf-8"))


def fetch_messages(channel_id: str, after_timestamp: str | None = None) -> list[dict]:
    """Fetch up to 100 messages from a channel, including thread messages."""
    params = {"limit": "100"}
    if after_timestamp:
        params["after"] = timestamp_to_snowflake(after_timestamp)
    query = urllib.parse.urlencode(params)
    messages = _discord_get(f"/channels/{channel_id}/messages?{query}")

    # Collect thread messages
    thread_ids = {msg["thread"]["id"] for msg in messages if msg.get("thread")}
    for tid in thread_ids:
        thread_params = {"limit": "100"}
        if after_timestamp:
            thread_params["after"] = timestamp_to_snowflake(after_timestamp)
        thread_query = urllib.parse.urlencode(thread_params)
        thread_msgs = _discord_get(f"/channels/{tid}/messages?{thread_query}")
        messages.extend(thread_msgs)

    return messages


def format_messages(messages: list[dict]) -> str:
    """Format messages as readable text, sorted by timestamp."""
    lines = []
    for msg in sorted(messages, key=lambda m: m["timestamp"]):
        content = msg.get("content", "").strip()
        if not content:
            continue
        author = msg["author"]["username"]
        ts = msg["timestamp"]
        lines.append(f"[{ts}] {author}: {content}")
    return "\n".join(lines)


def fetch_channel_threads(guild_id: str) -> list[dict]:
    """Fetch all active threads in a guild."""
    result = _discord_get(f"/guilds/{guild_id}/threads/active")
    return result.get("threads", [])


def get_image_urls(message: dict) -> list[str]:
    """Extract image attachment URLs from a Discord message."""
    attachments = message.get("attachments", [])
    return [
        a["url"]
        for a in attachments
        if a.get("content_type", "").startswith("image/")
    ]
