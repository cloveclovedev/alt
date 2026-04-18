# X Post Draft Optimization — Design Spec

Improve X post draft generation and posting skills based on 2026 X algorithm best practices, maximizing post visibility for @cloveclovedev.

## Background

The x-draft-cloud and x-post-cloud skills are functional but were designed without consideration for X algorithm optimization. Research into 2026 X algorithm behavior reveals several actionable improvements:

- Links in main posts cause 30-90% reach suppression → must use self-reply
- Posts with images get ~2x engagement boost
- Hashtag count above 2 triggers spam signals
- Post type diversity (progress, technical, problem-solution, reflection) drives different engagement signals (bookmarks, replies, profile clicks)
- Design docs are an untapped source of high-quality "technical decision" content

Related issues:
- #68 — Auto-generate code diff images (future)
- #69 — PR pain point auto-documentation (future)
- #70 — Metrics sharing posts (future, post-launch)
- #71 — alt repository OSS migration

### Goals

- Improve draft quality by diversifying post types and following algorithm-aware rules
- Eliminate link penalty by moving links to self-replies
- Add design docs as a content source for technical posts
- Support draft stockpiling (generate multiple drafts, post over subsequent days)
- Prepare data model for future enhancements (#68, #69)

### Non-Goals

- X Premium subscription (user preference)
- Automated reply to other users' comments
- Code diff image generation (#68)
- PR pain point capture (#69)

## Architecture Overview

```
x-draft-cloud (10:00/18:00 JST)
    |
    v
Data Collection (parallel)
    |-- GitHub: commits, merged PRs, closed issues (existing)
    |-- Discord: memos (existing)
    |-- Discord: daily plan (existing)
    |-- NEW: design docs (docs/superpowers/specs/)
    |
    v
Draft Generation (improved)
    |-- Classify post type (progress/technical/problem-solution/reflection)
    |-- Select hashtags (max 2, tech-name preferred)
    |-- Determine reply_link from product mapping
    |-- Compare against existing x_draft entries to avoid duplication
    |-- Generate 1-3 drafts
    |
    v
Save & Notify
    |-- alt-db: save with extended metadata
    +-- alt-discord: post to journal channel

x-post-cloud (hourly via cloud-scheduler)
    |
    v
Post approved drafts
    |-- Post main tweet (text + hashtags + image if available)
    |-- NEW: Post self-reply with link (if reply_link set)
    +-- Notify Discord
```

## Changes

### 1. Rename tone-guide to x-post-guide and add new sections

Rename `data/content/tone-guide.md` → `data/content/x-post-guide.md`. The file originally covered only tone/style (口調), but this spec adds post type classification and algorithm-aware rules that go beyond tone. The new name reflects the file's expanded scope as a comprehensive X post writing guide. Update the reference in `x-draft-cloud/SKILL.md` accordingly.

Add the following sections to the renamed file:

**Post type classification:**

| Type | Focus | Example |
|---|---|---|
| progress | What changed + user impact | 「peppercheckに通知機能を追加した。設定画面からON/OFFを選べる」 |
| technical | What was chosen + why | 「状態管理にRiverpodを選んだ。Provider多すぎ問題がなくて見通しがいい」 |
| problem-solution | What broke + how it was fixed | 「Neonの接続プールでハマった。pgbouncerのprepared statement設定が原因だった」 |
| reflection | Insight or observation | 「OSSでやると設計の意図をドキュメントに残す習慣がつく」 |

**Algorithm-aware rules:**

- Never include links in the main post body. Links go in self-replies only.
- Maximum 2 hashtags per post. Prefer specific tech names (e.g., `#Flutter`, `#Riverpod`) over generic ones. Use `#個人開発` or `#OSS` for reflection-type posts.
- Do not disclose private repository internals (DB schema, API design, internal architecture). Focus on decisions and reasoning, not implementation details.
- When sourcing from design docs, focus on the "why" of technical choices, not the "how" of implementation.
- If an image is available (PR screenshot, UI capture), always attach it.

**Post structure guidance:**

```
progress:    何をしたか + ユーザーにとって何が変わるか
technical:   何を選んだか + なぜそれにしたか（1-2行で）
problem-solution: 何にハマったか + どう解決したか
reflection:  気づき・考察（短く、余韻を残す）
```

### 2. x-draft-cloud skill changes (`.claude/skills/x-draft-cloud/SKILL.md`)

#### 2a. New data source: design docs

Add to Phase 2 (Data Collection):

- Scan `docs/superpowers/specs/*-design.md` in the alt repo (local filesystem)
- Design docs in other repos are not scanned (they may not exist or follow a different structure). If needed in the future, add per-repo doc paths to `alt.toml`.
- Retrieve all existing `x_draft` entries: `uv run alt-db --json entry list --type x_draft`
- For each design doc, compare against existing draft contents to identify angles not yet covered
- Design docs with untapped content become source material for `technical` type posts

#### 2b. Post type classification

Add to Phase 4 (Draft Generation):

After collecting all material, classify each potential draft into a post type:

- Merged PR with user-visible changes → `progress`
- Design doc with technical decision → `technical`
- PR comment with `<!-- x-draft-source -->` marker → `problem-solution` (future, #69)
- Discord memo with insight/reflection → `reflection`
- General development activity → `progress` (default)

#### 2c. Hashtag selection

Add hashtag selection logic:

1. Identify the primary technology mentioned in the content (Flutter, Riverpod, Next.js, PostgreSQL, Neon, etc.)
2. For `progress` / `technical` / `problem-solution`: use tech name hashtag + optionally `#OSS`
3. For `reflection`: use `#個人開発` or `#OSS`
4. Maximum 2 hashtags total

#### 2d. Reply link determination

Read product link mapping from `alt.toml` (`[x.product_links]`). Based on the draft's tags (project association), auto-assign `reply_link`.

Fallback priority:
1. Product website (if tagged project has one)
2. Source PR URL (if draft originated from a specific PR)
3. GitHub repository URL
4. null (no self-reply)

#### 2e. Extended metadata

Update saved draft metadata format:

```json
{
  "source_commits": ["repo:hash", ...],
  "source_memo_count": 3,
  "source_design_doc": "2026-04-08-x-draft-automation-design.md",
  "source_pr_url": "https://github.com/cloveclovedev/peppercheck/pull/42",
  "generated_at": "2026-04-12T10:00:00+09:00",
  "image_url": null,
  "post_type": "technical",
  "hashtags": ["#Flutter", "#OSS"],
  "reply_link": "https://peppercheck.dev",
  "reply_link_label": "詳細はこちら"
}
```

New fields:
- `post_type` — draft classification
- `hashtags` — selected hashtags (array, max 2)
- `reply_link` — URL for self-reply (null if none)
- `reply_link_label` — display text for self-reply (default: "詳細はこちら")
- `source_design_doc` — filename of source design doc (if applicable)
- `source_pr_url` — URL of source PR (if applicable)

These fields are additive. Existing fields remain unchanged for backward compatibility.

#### 2f. Draft content format

The draft `content` field now includes hashtags at the end:

```
peppercheckに通知機能を追加した。設定画面からON/OFFを選べる

#Flutter #OSS
```

This ensures hashtags are visible during review on the webapp and posted as part of the tweet text.

#### 2g. Multiple drafts from design docs

When a design doc has multiple postable angles (e.g., tech choice rationale, architecture overview, trade-off analysis), generate separate drafts for each angle. Each draft is saved as its own entry with `source_design_doc` referencing the same file. The deduplication logic (comparing against existing x_draft content) prevents re-generating previously created angles.

### 3. x-post-cloud skill changes (`.claude/skills/x-post-cloud/SKILL.md`)

#### 3a. Self-reply with link

After successfully posting the main tweet, if `metadata.reply_link` is not null:

1. Post a reply tweet using `reply.in_reply_to_tweet_id` set to the main tweet's ID
2. Reply content: `{reply_link_label}\n{reply_link}`
3. Save the reply tweet ID in metadata as `reply_tweet_id`

```python
# Main tweet
body = {"text": tweet_text}
# ... post and get tweet_id

# Self-reply (if reply_link exists)
if reply_link:
    reply_body = {
        "text": f"{reply_link_label}\n{reply_link}",
        "reply": {"in_reply_to_tweet_id": tweet_id}
    }
    # ... post reply
```

#### 3b. Updated entry metadata after posting

```json
{
  "tweet_id": "1234567890",
  "reply_tweet_id": "1234567891",
  "posted_at": "2026-04-12T12:03:00+09:00"
}
```

#### 3c. Error handling for self-reply

If the main tweet succeeds but self-reply fails:
- Log error to Discord
- Keep status as `posted` (main tweet was successful)
- Include error note in metadata: `"reply_error": "..."`
- Do not retry — the main tweet is already live

### 4. alt.toml changes

Add product link mapping:

```toml
[x.product_links]
peppercheck = "https://peppercheck.dev"
alt = "https://github.com/mkuri/alt"
cloveclove-site = "https://cloveclove.dev"
cloveclove-developer-docs = "https://docs.cloveclove.dev"
```

### 5. Webapp changes (`webapp/src/components/posts/draft-card.tsx`)

Minor UI changes to make approval decisions easier:

#### 5a. Move tags above action buttons

Currently tags are displayed at the bottom of the card, below Approve/Skip buttons. Move them above the action buttons so they are visible before deciding.

#### 5b. Display reply_link

Show `metadata.reply_link` as a small preview below the tags, so the reviewer can verify the link is correct before approving:

```
[Flutter] [OSS]                    ← tags (moved up)
🔗 https://peppercheck.dev         ← reply_link preview
[Approve] [Skip]                   ← action buttons
```

If `reply_link` is null, show nothing (no empty state needed).

#### 5c. Display post_type badge

Show `metadata.post_type` as a badge in the header row next to the status badge:

```
Draft | technical                  2026-04-12 10:00
```

This helps the reviewer understand the intent of the draft at a glance.

## Future Extension Points

These are designed to plug into the architecture without structural changes:

| Issue | Extension Point |
|---|---|
| #68 Code diff images | Add image generation step between data collection and draft generation. Set `image_url` in metadata. No other changes needed. |
| #69 PR pain points | Add PR comment scanning to data collection (Phase 2). Look for `<!-- x-draft-source -->` markers. Classify as `problem-solution` type. |
| #70 Metrics sharing | Add new post type `metrics`. New data source: app store / analytics API. Same metadata structure. |
| #71 alt OSS | Remove "private repo internals" guard from x-post-guide. Enable design doc scanning for alt repo. |

## Testing

- **x-post-guide:** Review generated drafts manually for compliance with new rules
- **x-draft-cloud:** Run manually, verify:
  - Design docs are scanned and compared against existing drafts
  - Post types are correctly classified
  - Hashtags follow the rules (max 2, tech-specific)
  - reply_link is correctly derived from alt.toml
  - Multiple drafts from a single design doc work
- **x-post-cloud:** Run manually with an approved draft, verify:
  - Main tweet posts successfully
  - Self-reply with link posts as a reply to the main tweet
  - Metadata is updated with both tweet IDs
  - Error in self-reply doesn't affect main tweet status
- **End-to-end:** Generate draft → approve on webapp → verify main tweet + self-reply appear on X
