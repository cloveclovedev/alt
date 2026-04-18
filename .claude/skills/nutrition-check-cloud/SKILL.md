---
name: nutrition-check-cloud
description: "[Cloud scheduled task] Track daily nutrition from Discord meal posts, provide periodic check-ins with calorie/protein progress"
---

# Nutrition Check (Cloud)

Autonomous skill for Cloud scheduled tasks. Reads meal posts from a daily Discord thread, estimates nutrition via LLM, and provides periodic check-ins with progress toward daily targets.

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
Use this to derive `<today>` (YYYY-MM-DD), `<yesterday>` (YYYY-MM-DD), and `<current_hour>` (HH).

Read `alt.toml` for configuration:
- `[nutrition] channel_id` — Discord channel for threads
- `[nutrition] protein_coefficient` — g per kg body weight (for target recalculation)
- `[nutrition] activity_factor` — activity multiplier (for target recalculation)
- `[nutrition] lean_bulk_surplus_kcal` — daily caloric surplus (for target recalculation)
- `[discord] daily_channel_id` — fallback if nutrition channel not set

Determine the current mode based on `<current_hour>`:
- **Hour 6** → `morning-summary`
- **Hour 10** → `check` (breakfast)
- **Hour 15** → `check` (lunch)
- **Hour 21** → `check` (dinner)
- **Hour 0** → `final-check` (uses yesterday's date)

### Phase 2: Morning Summary (mode: morning-summary)

**Only runs at 06:00.**

#### 2a: Yesterday's Summary

Get yesterday's nutrition data:
```bash
uv run alt-db --json nutrition-log summary --date <yesterday>
uv run alt-db --json nutrition-log list --date <yesterday>
```

Get yesterday's target:
```bash
uv run alt-db --json nutrition-target active
```

Compose the morning summary in this format:
```
📊 <yesterday> 栄養サマリー

🔥 カロリー: <actual> / <target>kcal (<percent>%)
💪 タンパク質: <actual> / <target>g (<percent>%)
💊 サプリ: Mg <✅/❌> VitD <✅/❌> K <✅/❌>

🍽 内訳:
  朝食: <items> (<cal>kcal, P<protein>g)
  昼食: ...
  夕食: ...
  ...

📝 評価: <brief evaluation of the day's nutrition — protein distribution,
balance, suggestions for improvement>
```

To check individual supplements, query logs with meal_type="supplement" for yesterday.

Post the summary to the **channel** (not thread):
```bash
uv run alt-discord post <channel_id> "<summary>"
```

#### 2b: Create Today's Thread

Get today's target:
```bash
uv run alt-db --json nutrition-target active
```

Create a new thread for today:
```bash
# Use Python to create the thread via Discord API
uv run python3 -c "
from alt_discord.poster import create_thread
result = create_thread(
    channel_id='<channel_id>',
    name='🍽 <today> 食事記録',
    initial_message='今日の目標: <calories>kcal / P: <protein>g',
)
print(result['id'])
"
```

Save the thread ID for later reference by storing it as an entry:
```bash
uv run alt-db entry add --type nutrition_thread \
  --title "Nutrition Thread <today>" \
  --content "<thread_id>" \
  --tags '["nutrition"]' \
  --metadata '{"date": "<today>", "thread_id": "<thread_id>"}'
```

### Phase 3: Check / Final Check (mode: check or final-check)

#### 3a: Find Today's Thread

For `final-check` (hour 0), use yesterday's date. Otherwise use today's date.

```bash
uv run alt-db --json entry list --type nutrition_thread --since 2d
```
Parse the entry metadata to find the thread_id for the target date.

If no thread found, skip this run (thread may not have been created yet).

#### 3b: Fetch Unprocessed Messages

Read messages from the thread:
```bash
uv run python3 -c "
from alt_discord.reader import fetch_messages
import json
msgs = fetch_messages('<thread_id>')
print(json.dumps(msgs))
"
```

For each message, check if already processed:
```bash
uv run alt-db --json nutrition-log check-message --message-id <message_id>
```

Skip messages that are already processed. Also skip messages from the bot itself (check `author.bot == true`).

#### 3c: Process Each New Message

For each unprocessed message:

1. **Extract content**: Get `message.content` (text) and image URLs via:
```bash
uv run python3 -c "
from alt_discord.reader import get_image_urls
import json
urls = get_image_urls(<message_json>)
print(json.dumps(urls))
"
```

2. **Check nutrition_items for known items**:
```bash
uv run alt-db --json nutrition-item list
```
Compare the message text against registered item names.

3. **Determine meal_type** from the current hour:
   - Hour 10 check → "breakfast"
   - Hour 15 check → "lunch"
   - Hour 21 check → "dinner"
   - Hour 0 check → "snack" (late night)
   - If the message explicitly mentions a meal type, use that instead.

4. **Process based on content type**:

   **Known item match** (e.g., "プロテイン"):
   - Use the registered nutrition values directly.
   - `estimated_by = "item_lookup"`

   **"サプリ" keyword**:
   - Create 3 separate log entries for each supplement (Dear-Natura Ca/Mg/Zn/VitD, Dear-Natura GOLD EPA&DHA, PURELAB カリウム).
   - Each with `meal_type="supplement"`, `supplement_taken=true`, `estimated_by="item_lookup"`.

   **Unknown text or image — use the following priority order:**

   **Priority 1: Nutrition label read**
   If any image contains a visible nutrition facts label (栄養成分表示), read the calorie and protein values directly from the label. Do not estimate — use the exact numbers shown.
   `estimated_by = "label_read"`

   **Priority 2: Web lookup of official values**
   If a brand name, store name, or specific product name is identifiable from the text or image, search the web for official nutrition information (e.g., `"<product name>" "<brand>" 栄養成分`). If an official value is found on the brand's website, product page, or reliable nutrition database, use it.
   `estimated_by = "web_lookup"`

   **Priority 3: Web-search-assisted LLM estimation**
   If neither a nutrition label nor a specific brand/product is identifiable:
   1. Identify the food item(s) from the text and/or image.
   2. Search the web for `"<food name>" カロリー 栄養素` to find standard nutritional values for similar dishes in Japan.
   3. Using the search results as reference and the photo's apparent portion size, estimate calories and protein.
   4. Always assume Japanese food portions and recipes (not American).
   5. When uncertain, estimate conservatively — do not overestimate.
   `estimated_by = "llm"`

   **Item registration request** (e.g., "プロテイン登録して。カロリー120、タンパク質24g"):
   - Parse the item name, calories, and protein from the message.
   - Register in nutrition_items:
   ```bash
   uv run alt-db nutrition-item add --name "<name>" --calories <cal> --protein <protein>
   ```
   - Reply in thread confirming registration.
   - Do NOT create a nutrition_log entry for registration requests.

   **Item update/delete request**:
   - Parse and execute the update/delete:
   ```bash
   uv run alt-db nutrition-item update --name "<name>" --calories <cal>
   uv run alt-db nutrition-item delete --name "<name>"
   ```
   - Reply in thread confirming the change.

5. **Save to nutrition_logs**:
```bash
uv run alt-db nutrition-log add \
  --date <target_date> \
  --meal-type <meal_type> \
  --description "<food_description>" \
  --calories <estimated_cal> \
  --protein <estimated_protein> \
  --source-message-id <message_id> \
  --estimated-by <method>
```

#### 3d: Post Check-in

Get daily summary:
```bash
uv run alt-db --json nutrition-log summary --date <target_date>
uv run alt-db --json nutrition-target active
```

**For interim check (10/15/21)**, post to thread:
```
⏰ HH:00 チェック

ここまで: <cal>kcal / P: <protein>g
残り目標: <remaining_cal>kcal / P: <remaining_protein>g

💡 <specific actionable suggestion based on remaining needs and time of day>
```

**For final check (00:00)**, post to thread:
```
🌙 最終チェック

今日の合計: <cal>kcal / P: <protein>g
目標比: カロリー <cal_percent>% / タンパク質 <protein_percent>%

⚡ <specific suggestion if targets not met, e.g., "プロテイン1杯(P23.0g)で目標達成です。飲んだら「プロテイン」と投稿してください。">
```

If targets are already met, congratulate:
```
🎉 今日の目標達成！お疲れさまでした。
```

Post the check-in to the thread:
```bash
uv run alt-discord post <thread_id> "<check_in_message>"
```

### Phase 4: Auto-Registration Proposal

During Phase 3c processing, track LLM-estimated food names. After processing all messages, check if any food name has appeared 3+ times in nutrition_logs with `estimated_by = "llm"`:

```bash
uv run python3 -c "
# Query for frequently LLM-estimated items
# SELECT description, COUNT(*), AVG(calories_kcal), AVG(protein_g)
# FROM nutrition_logs WHERE estimated_by = 'llm'
# GROUP BY description HAVING COUNT(*) >= 3
"
```

For each qualifying item, post a registration proposal to the thread:
```
💡 "<food_name>"をよく食べていますね。平均値(<avg_cal>kcal, P<avg_protein>g)で登録しますか？
「<food_name>登録して。カロリー<cal>、タンパク質<protein>g」と投稿してください。
```

### Environment Variables (set in Cloud environment)

- `DISCORD_BOT_TOKEN` — Discord bot token
- `NEON_HOST` — Neon PostgreSQL host
- `NEON_DATABASE` — Neon database name
- `NEON_USER` — Neon user
- `NEON_PASSWORD` — Neon password
