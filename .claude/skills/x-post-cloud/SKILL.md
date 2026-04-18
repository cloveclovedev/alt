---
name: x-post-cloud
description: "[Cloud scheduled task] Post approved X drafts that are past their scheduled time"
---

# X Post Publisher

Autonomous skill for Cloud scheduled tasks. Checks for approved X drafts past their scheduled time and posts them via X API v2.

## When invoked

### Phase 0: Environment

Install dependencies:
```bash
uv sync
```

### Phase 1: Check for Drafts Due

Get the current JST time:
```bash
TZ=Asia/Tokyo date '+%Y-%m-%dT%H:%M:%S+09:00'
```

List approved x_draft entries:
```bash
uv run alt-db --json entry list --type x_draft --status approved
```

Filter entries where `metadata.scheduled_at <= current_time`. If none are due, log "No drafts due for posting" and end the session.

### Phase 2: Post Each Draft

Read `alt.toml` for `[discord.content] draft_channel_id`.

For each draft due for posting:

#### 2a: Upload Image (if present)

If `metadata.image_url` exists, download the image and upload to X:

```bash
# Download image
curl -sL "<image_url>" -o /tmp/x_image

# Upload to X media endpoint (OAuth 1.0a signed)
curl -s -X POST "https://upload.x.com/2/media/upload" \
  -H "Authorization: OAuth oauth_consumer_key=\"$X_CONSUMER_KEY\", oauth_token=\"$X_ACCESS_TOKEN\", oauth_signature_method=\"HMAC-SHA256\", oauth_timestamp=\"...\", oauth_nonce=\"...\", oauth_version=\"1.0\", oauth_signature=\"...\"" \
  -F "media=@/tmp/x_image" \
  -F "media_category=tweet_image"
```

Note: OAuth signature generation is complex. Use a Python one-liner or small script to generate the signed request. Example:

```bash
python3 -c "
import os, json, hmac, hashlib, base64, time, urllib.parse, urllib.request

consumer_key = os.environ['X_CONSUMER_KEY']
consumer_secret = os.environ['X_CONSUMER_SECRET']
access_token = os.environ['X_ACCESS_TOKEN']
access_secret = os.environ['X_ACCESS_TOKEN_SECRET']

def oauth_sign(method, url, params, consumer_secret, token_secret):
    base = '&'.join([method.upper(), urllib.parse.quote(url, safe=''), urllib.parse.quote('&'.join(f'{k}={urllib.parse.quote(str(v), safe=\"\")}' for k, v in sorted(params.items())), safe='')])
    key = f'{urllib.parse.quote(consumer_secret)}&{urllib.parse.quote(token_secret)}'
    return base64.b64encode(hmac.new(key.encode(), base.encode(), hashlib.sha256).digest()).decode()

import sys
text = sys.argv[1]
media_id = sys.argv[2] if len(sys.argv) > 2 else None

url = 'https://api.x.com/2/tweets'
body = {'text': text}
if media_id:
    body['media'] = {'media_ids': [media_id]}

params = {
    'oauth_consumer_key': consumer_key,
    'oauth_nonce': base64.b64encode(os.urandom(32)).decode().strip('=+/'),
    'oauth_signature_method': 'HMAC-SHA256',
    'oauth_timestamp': str(int(time.time())),
    'oauth_token': access_token,
    'oauth_version': '1.0',
}
params['oauth_signature'] = oauth_sign('POST', url, params, consumer_secret, access_secret)

auth_header = 'OAuth ' + ', '.join(f'{k}=\"{urllib.parse.quote(str(v), safe=\"\")}\"' for k, v in sorted(params.items()))

req = urllib.request.Request(url, data=json.dumps(body).encode(), headers={'Authorization': auth_header, 'Content-Type': 'application/json'}, method='POST')
resp = urllib.request.urlopen(req)
result = json.loads(resp.read())
print(json.dumps(result))
" "<tweet_text>" "<media_id_or_empty>"
```

Save the tweet ID from the response (`data.id`).

#### 2b: Post Self-Reply with Link (if present)

If `metadata.reply_link` is not null, post a reply to the main tweet:

```bash
python3 -c "
import os, json, hmac, hashlib, base64, time, urllib.parse, urllib.request

consumer_key = os.environ['X_CONSUMER_KEY']
consumer_secret = os.environ['X_CONSUMER_SECRET']
access_token = os.environ['X_ACCESS_TOKEN']
access_secret = os.environ['X_ACCESS_TOKEN_SECRET']

def oauth_sign(method, url, params, consumer_secret, token_secret):
    base = '&'.join([method.upper(), urllib.parse.quote(url, safe=''), urllib.parse.quote('&'.join(f'{k}={urllib.parse.quote(str(v), safe=\"\")}' for k, v in sorted(params.items())), safe='')])
    key = f'{urllib.parse.quote(consumer_secret)}&{urllib.parse.quote(token_secret)}'
    return base64.b64encode(hmac.new(key.encode(), base.encode(), hashlib.sha256).digest()).decode()

import sys
reply_text = sys.argv[1]
in_reply_to = sys.argv[2]

url = 'https://api.x.com/2/tweets'
body = {'text': reply_text, 'reply': {'in_reply_to_tweet_id': in_reply_to}}

params = {
    'oauth_consumer_key': consumer_key,
    'oauth_nonce': base64.b64encode(os.urandom(32)).decode().strip('=+/'),
    'oauth_signature_method': 'HMAC-SHA256',
    'oauth_timestamp': str(int(time.time())),
    'oauth_token': access_token,
    'oauth_version': '1.0',
}
params['oauth_signature'] = oauth_sign('POST', url, params, consumer_secret, access_secret)

auth_header = 'OAuth ' + ', '.join(f'{k}=\"{urllib.parse.quote(str(v), safe=\"\")}\"' for k, v in sorted(params.items()))

req = urllib.request.Request(url, data=json.dumps(body).encode(), headers={'Authorization': auth_header, 'Content-Type': 'application/json'}, method='POST')
resp = urllib.request.urlopen(req)
result = json.loads(resp.read())
print(json.dumps(result))
" "<reply_link_label>
<reply_link>" "<main_tweet_id>"
```

Where `<reply_link_label>` and `<reply_link>` come from the draft's metadata (e.g., `詳細はこちら\nhttps://peppercheck.dev`), and `<main_tweet_id>` is the `data.id` from the main tweet response.

Save the reply tweet ID from the response (`data.id`).

#### 2c: Update Entry Status

```bash
uv run alt-db entry update <id> --status posted \
  --metadata '{"tweet_id": "<tweet_id>", "reply_tweet_id": "<reply_tweet_id_or_null>", "posted_at": "<current_jst_iso>"}'
```

#### 2d: Notify Discord

```bash
uv run alt-discord post <draft_channel_id> "Posted to X: https://x.com/i/status/<tweet_id>"
```

### Phase 3: Error Handling

If X API returns an error for a draft:
1. Log the error to Discord: `uv run alt-discord post <channel_id> "X post failed for <entry_id>: <error>"`
2. Keep status as `approved` (will retry next hour)
3. Continue with remaining drafts

If the main tweet succeeds but self-reply fails:
1. Log the error to Discord: `uv run alt-discord post <channel_id> "Self-reply failed for <entry_id>: <error>"`
2. Update entry status to `posted` (main tweet was successful)
3. Include error note in metadata: `"reply_error": "<error_message>"`
4. Do not retry — the main tweet is already live

### Environment Variables (set in Cloud environment)

- `DISCORD_BOT_TOKEN` — Discord bot token
- `DATABASE_URL` — Neon PostgreSQL connection string
- `X_CONSUMER_KEY` — X API OAuth 1.0a consumer key
- `X_CONSUMER_SECRET` — X API OAuth 1.0a consumer secret
- `X_ACCESS_TOKEN` — X API OAuth 1.0a access token
- `X_ACCESS_TOKEN_SECRET` — X API OAuth 1.0a access token secret
