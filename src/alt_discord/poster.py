"""Post messages to a Discord channel via Bot API."""

import json
import os
import urllib.request

DISCORD_API = "https://discord.com/api/v10"
MAX_LENGTH = 2000


def split_message(text: str) -> list[str]:
    """Split text into chunks that fit Discord's 2000 char limit."""
    if len(text) <= MAX_LENGTH:
        return [text]

    chunks = []
    current = ""
    for line in text.split("\n"):
        if len(current) + len(line) + 1 > MAX_LENGTH:
            if current:
                chunks.append(current)
            current = line[:MAX_LENGTH]
        else:
            current = f"{current}\n{line}" if current else line
    if current:
        chunks.append(current)
    return chunks


def post_message(channel_id: str, content: str) -> list[str]:
    """Post a message to a Discord channel. Returns list of message IDs."""
    token = os.environ["DISCORD_BOT_TOKEN"]
    message_ids = []

    for chunk in split_message(content):
        data = json.dumps({"content": chunk}).encode()
        req = urllib.request.Request(
            f"{DISCORD_API}/channels/{channel_id}/messages",
            data=data,
            headers={
                "Authorization": f"Bot {token}",
                "Content-Type": "application/json",
                "User-Agent": "DiscordBot (https://github.com/cloveclovedev/alt, 0.1.0)",
            },
            method="POST",
        )
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read().decode("utf-8"))
            message_ids.append(result.get("id", ""))

    return message_ids


def create_thread(channel_id: str, name: str, initial_message: str) -> dict:
    """Create a new thread in a channel with an initial message.

    Uses the 'Start Thread without Message' endpoint.
    Returns the created thread object (includes 'id' and 'name').
    """
    token = os.environ["DISCORD_BOT_TOKEN"]
    data = json.dumps({
        "name": name,
        "auto_archive_duration": 1440,  # 24 hours
        "type": 11,  # PUBLIC_THREAD
        "message": {"content": initial_message},
    }).encode()
    req = urllib.request.Request(
        f"{DISCORD_API}/channels/{channel_id}/threads",
        data=data,
        headers={
            "Authorization": f"Bot {token}",
            "Content-Type": "application/json",
            "User-Agent": "DiscordBot (https://github.com/cloveclovedev/alt, 0.1.0)",
        },
        method="POST",
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode("utf-8"))


def create_thread_from_message(
    channel_id: str, message_id: str, name: str
) -> dict:
    """Create a thread on an existing message.

    Uses the 'Start Thread from Message' endpoint.
    Returns the created thread object (includes 'id' and 'name').
    """
    token = os.environ["DISCORD_BOT_TOKEN"]
    data = json.dumps({
        "name": name,
        "auto_archive_duration": 1440,  # 24 hours
    }).encode()
    req = urllib.request.Request(
        f"{DISCORD_API}/channels/{channel_id}/messages/{message_id}/threads",
        data=data,
        headers={
            "Authorization": f"Bot {token}",
            "Content-Type": "application/json",
            "User-Agent": "DiscordBot (https://github.com/cloveclovedev/alt, 0.1.0)",
        },
        method="POST",
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read().decode("utf-8"))
