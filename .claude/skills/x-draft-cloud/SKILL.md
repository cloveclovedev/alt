---
name: x-draft-cloud
description: "[Cloud scheduled task] Generate X post drafts from development activity and Discord memos, post to Discord for review"
---

# X Post Draft Generation

Autonomous skill for Cloud scheduled tasks. Collects development activity, generates X post drafts, and posts them to Discord for review.

## When invoked

### Phase 0: Environment

Install dependencies:
```bash
uv sync
```

### Phase 1: Setup

Determine the current date/time in JST:
```bash
TZ=Asia/Tokyo date '+%Y-%m-%dT%H:%M:%S+09:00'
```

Get the timestamp of the last draft from the entries table:
```bash
uv run alt-db --json entry list --type x_draft 2>/dev/null | python3 -c "
import sys, json
entries = json.load(sys.stdin)
if entries:
    print(entries[0]['created_at'])
" 2>/dev/null
```
If empty (no previous draft), use 24 hours ago as the lookback boundary.

Read configuration via:
- `uv run alt-db config get plan.github.repos` — repos to scan
- `uv run alt-db config get plan.discord.channel_id` — daily plan channel (for context lookup)
- `uv run alt-db config get draft.x.discord.input_channel_ids` — memo source channels (array)
- `uv run alt-db config get draft.x.discord.channel_id` — draft output channel (journal)
- `uv run alt-db config get draft.x.product_links` — product name to URL mapping for self-reply links

### Phase 2: Data Collection

Run these in parallel:

1. **GitHub Activity (since last draft):**
   For each repo in `plan.github.repos`:
   ```bash
   gh api "/repos/{owner}/{repo}/commits?since=<last_draft_time>&per_page=20" --jq '.[].commit.message'
   ```
   ```bash
   gh api "/repos/{owner}/{repo}/pulls?state=closed&sort=updated&direction=desc&per_page=10" --jq '.[] | select(.merged_at != null) | select(.merged_at > "<last_draft_time>") | "#\(.number) \(.title)"'
   ```
   ```bash
   gh issue list --repo {owner}/{repo} --state closed --json number,title,closedAt --jq '.[] | select(.closedAt > "<last_draft_time>") | "#\(.number) \(.title)"'
   ```

   Also extract screenshot images from merged PR bodies:
   ```bash
   gh api "/repos/{owner}/{repo}/pulls?state=closed&sort=updated&direction=desc&per_page=10" \
     --jq '.[] | select(.merged_at != null) | select(.merged_at > "<last_draft_time>") | .body' \
     | grep -oE 'https://github\.com/user-attachments/assets/[a-zA-Z0-9-]+' || true
   ```
   If images are found, include the first relevant image URL when saving the draft.

2. **Discord Memos (since last draft):**
   For each `<channel_id>` in `draft.x.discord.input_channel_ids` (array — may contain one or more channels):
   ```bash
   uv run alt-discord read <channel_id> --after <last_draft_time>
   ```

3. **Today's Daily Plan:**
   ```bash
   uv run alt-discord read <daily_channel_id> --after <today_start_iso>
   ```

4. **Design Docs (all time):**
   List all design doc files:
   ```bash
   ls docs/superpowers/specs/*-design.md 2>/dev/null
   ```
   Read each design doc's content. These are evaluated against existing drafts in Phase 3.

5. **Existing X Drafts (for deduplication):**
   ```bash
   uv run alt-db --json entry list --type x_draft 2>/dev/null
   ```
   This is used in Phase 4 to avoid generating drafts that cover the same angle as existing entries.

### Phase 3: Evaluation

Evaluate the collected data, including design docs. If there is no noteworthy material from any source (GitHub activity, Discord memos, or untapped design doc angles):
```bash
uv run alt-discord post <draft_channel_id> "X draft: no material, skipping"
```
Then end the session.

### Phase 4: Draft Generation

If there is material, read `x-post-guide.md` in this skill directory for style guidelines if it exists. Otherwise, see `x-post-guide.example.md` as a template — copy it to `x-post-guide.md` and adapt to your own voice.

**Classify post type** for each potential draft:
- Merged PR with user-visible changes → `progress`
- Design doc with technical decision → `technical`
- PR comment with `<!-- x-draft-source -->` marker → `problem-solution` (future, see #69)
- Discord memo with insight/reflection → `reflection`
- General development activity → `progress` (default)

**Select hashtags** (maximum 2):
1. Identify the primary technology mentioned in the content (Flutter, Riverpod, Next.js, PostgreSQL, Neon, etc.)
2. For `progress` / `technical` / `problem-solution`: use tech name hashtag + optionally `#OSS`
3. For `reflection`: use `#個人開発` or `#OSS`

**Determine reply_link** from `uv run alt-db config get draft.x.product_links`:
1. Product website (if the draft's primary tag matches a key in `product_links`)
2. Source PR URL (if the draft originated from a specific merged PR)
3. GitHub repository URL (from `plan.github.repos`)
4. null (no self-reply — use this if no relevant link exists)

**Deduplicate against existing drafts:**
- Compare each potential draft angle against the content of existing `x_draft` entries (retrieved in Phase 2, step 5)
- Skip angles that are substantially similar to existing drafts
- For design docs: multiple drafts from different angles of the same doc are fine, but don't regenerate an angle already covered

**Generate 1-3 draft options:**
- Follow the tone guide strictly
- Convert technical activity to user-facing language
- Keep it concise — do not pad short content
- Write in Japanese
- Include hashtags at the end of the draft text (separated by a blank line)
- Stay within 280 characters per draft (including hashtags)

### Phase 5: Save and Post

Save each draft to the entries table:
```bash
uv run alt-db entry add --type x_draft --status draft \
  --title "<one_line_summary>" \
  --content "<draft_text_with_hashtags>" \
  --metadata '{"source_commits": [...], "source_memo_count": <n>, "source_design_doc": "<filename_or_null>", "source_pr_url": "<url_or_null>", "generated_at": "<iso_timestamp>", "image_url": "<url_or_null>", "post_type": "<progress|technical|problem-solution|reflection>", "hashtags": ["#Tech", "#OSS"], "reply_link": "<url_or_null>", "reply_link_label": "詳細はこちら", "project": "<relevant_project>"}'
```

Where:
- `post_type`: the classified type from Phase 4
- `hashtags`: array of 1-2 hashtags selected in Phase 4
- `reply_link`: URL determined in Phase 4 (null if none)
- `reply_link_label`: default to "詳細はこちら"
- `source_design_doc`: filename of the source design doc (null if not from a design doc)
- `source_pr_url`: URL of the source PR (null if not from a PR)

Post the drafts to Discord:
```bash
uv run alt-discord post <draft_channel_id> "$(cat <<'MSG'
X draft

Option 1: <draft text>

Option 2: <draft text>
MSG
)"
```

### Environment Variables (set in Cloud environment)

- `DISCORD_BOT_TOKEN` — Discord bot token
- `DATABASE_URL` — Neon PostgreSQL connection string
- `GH_TOKEN` — GitHub Personal Access Token (repo read)
