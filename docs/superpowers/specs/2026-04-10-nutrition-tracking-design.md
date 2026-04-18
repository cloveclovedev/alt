# Nutrition Tracking Design

## Overview

Daily nutrition tracking system integrated with Discord and the existing body composition goal framework. Tracks calories and protein intake to support a lean bulk strategy (increase skeletal muscle while keeping weight gain minimal).

## Goals

- Track daily calorie and protein intake from Discord meal posts (photos + text)
- Auto-calculate nutrition targets from body composition goals
- Provide periodic check-ins and actionable suggestions throughout the day
- Register frequently consumed items for quick lookup
- Track supplement intake (yes/no per day)

## Architecture

Single cloud skill (`nutrition-check-cloud`) scheduled at 5 time slots, with time-based behavior branching. Leverages existing `alt-discord` read/post infrastructure and Neon PostgreSQL.

```
Discord Thread (daily)          Cloud Skill              Database
┌─────────────────┐    ┌──────────────────────┐    ┌──────────────────┐
│ User posts meals │───>│ nutrition-check-cloud │───>│ nutrition_logs   │
│ (photo/text)     │    │                      │    │ nutrition_items   │
│                  │<───│ Check-in replies      │    │ nutrition_targets │
└─────────────────┘    └──────────────────────┘    └──────────────────┘
                              │                           │
                              │ LLM Vision (photos)       │
                              │ Item lookup (text)        │
                              v                           v
                        ┌──────────────┐          ┌──────────────────┐
                        │ body_measure- │          │ body_measurement │
                        │ ments (latest)│          │ _goals (active)  │
                        └──────────────┘          └──────────────────┘
```

## Database Schema

### `nutrition_items` (frequently consumed items dictionary)

| Column | Type | Description |
|--------|------|-------------|
| id | serial PK | |
| name | text, unique | Display name (e.g., "プロテイン") |
| calories_kcal | decimal | Calories per serving |
| protein_g | decimal(5,1) | Protein per serving (1 decimal place) |
| source | text | "user_registered" or "llm_estimated" |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### `nutrition_logs` (per-meal records)

| Column | Type | Description |
|--------|------|-------------|
| id | serial PK | |
| logged_date | date | Date in JST |
| meal_type | text | "breakfast", "lunch", "dinner", "snack", "supplement" |
| description | text | Meal content text (LLM summary included) |
| calories_kcal | decimal, nullable | |
| protein_g | decimal(5,1), nullable | |
| supplement_taken | boolean, default false | |
| source_message_id | text | Discord message ID (dedup key) |
| estimated_by | text | "llm", "item_lookup", "user" |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### `nutrition_targets` (daily targets with history)

| Column | Type | Description |
|--------|------|-------------|
| id | serial PK | |
| calories_kcal | decimal | Target daily calories |
| protein_g | decimal(5,1) | Target daily protein |
| effective_from | date | Start date |
| effective_until | date, nullable | End date (null = currently active) |
| rationale | text | Calculation basis (e.g., "73kg x 2.0 = 146g") |
| created_at | timestamptz | |
| updated_at | timestamptz | |

## Configuration

Added to `alt.toml`:

```toml
[nutrition]
channel_id = "YOUR_NUTRITION_CHANNEL_ID"    # Discord channel for threads
protein_coefficient = 2.0             # g per kg body weight
activity_factor = 1.5                 # sedentary=1.2, moderate=1.5, active=1.75
lean_bulk_surplus_kcal = 200          # daily caloric surplus
```

## Cloud Skill: `nutrition-check-cloud`

### Schedule & Behavior

| Time | Mode | Action |
|------|------|--------|
| 06:00 | morning-summary | Summarize yesterday's nutrition, post to **channel**. Create today's thread. |
| 10:00 | check-breakfast | Process new messages, post interim check to **thread** |
| 15:00 | check-lunch | Process new messages, post interim check to **thread** |
| 21:00 | check-dinner | Process new messages, post interim check to **thread** |
| 00:00 | final-check | Process new messages from **previous calendar day's thread**, post final check with last-minute suggestions to **thread** |

### Thread Management

- Created at 06:00 as part of morning-summary
- Thread title format: `🍽 2026-04-10 食事記録`
- Opening message includes daily targets: `今日の目標: 2,450kcal / P: 146.0g`
- Thread lookup: search channel threads by date string in title (e.g., "2026-04-09" for yesterday's summary)
- 00:00 final-check targets the previous calendar day's thread (the day that just ended)

### Message Processing Flow

```
For each unprocessed message in today's thread:
  1. Check source_message_id against nutrition_logs (skip if already processed)
  2. Classify message content:
     a. Text only → search nutrition_items by name
        - Match found → use registered values
        - No match → LLM estimates from text
     b. Image present → LLM Vision analyzes food photo
     c. Text + Image → LLM uses both for estimation
  3. Parse multiple items from single message ("プロテインとバナナ")
  4. Save to nutrition_logs with appropriate estimated_by value
  5. Handle supplement keywords:
     "サプリ" → create 3 separate nutrition_log entries (one per supplement)
     with meal_type="supplement" and each supplement's name as description.
     Individual supplement names (e.g., "カリウム") → single entry.
```

### nutrition_items Lookup

```
Priority:
1. Exact match: "プロテイン" → nutrition_items.name = "プロテイン"
2. Partial/fuzzy: "いつものプロテイン" → LLM extracts "プロテイン" → match
3. No match → LLM estimates nutrition from text/image
```

### Auto-Registration Proposal

When LLM estimates the same food item 3+ times, the skill proposes registering it:
> "オイコス"をよく食べていますね。平均値(69kcal, P10.0g)で登録しますか？

### Item Management via Discord

Users can manage nutrition_items through natural language in the thread:
- `プロテイン登録して。カロリー120、タンパク質24g` → insert
- `プロテインのカロリー130に変更して` → update
- `オイコスの登録消して` → delete

## Target Auto-Calculation

### Formula

```
Protein target:
  body_measurement_goals[weight_kg].target_value × nutrition.protein_coefficient
  Example: 73kg × 2.0 = 146.0g

Calorie target:
  body_measurements[latest].basal_metabolic_rate × nutrition.activity_factor
  + nutrition.lean_bulk_surplus_kcal
  Example: 1,650 × 1.5 + 200 = 2,675kcal
```

### Recalculation Trigger

- Detected when new InBody data shows ±1kg weight change from last target calculation
- Proposes new targets in thread; does NOT auto-update (requires user confirmation)

## Preset Items

| Name | Calories | Protein | Notes |
|------|----------|---------|-------|
| プロテイン | 249kcal | 23.0g | Custom mix: おなかにやさしいホエイプロテイン 濃厚リッチチョコ風味 + マルトデキストリン20g + クレアチン + 難消化性デキストリン + はちみつ~10g + シナモン |
| EAA | 50kcal | 6.2g | Custom mix: EAA ソルティライチ風味 + クレアチン |
| サプリ | 0kcal | 0g | Bundle: triggers recording of all 3 supplements below |
| Dear-Natura Ca/Mg/Zn/VitD | 0kcal | 0g | supplement_taken=true |
| Dear-Natura GOLD EPA&DHA | 0kcal | 0g | supplement_taken=true |
| PURELAB カリウム | 0kcal | 0g | supplement_taken=true |

## Output Formats

### Morning Summary (channel post)

```
📊 2026-04-09 栄養サマリー

🔥 カロリー: 2,380 / 2,675kcal (89%)
💪 タンパク質: 138.0 / 146.0g (95%)
💊 サプリ: Mg ✅ VitD ✅ K ❌

🍽 内訳:
  朝食: プロテイン, バナナ (220kcal, P26.0g)
  昼食: 鶏むね弁当 (650kcal, P42.0g)
  間食: オイコス (69kcal, P10.0g)
  夕食: 牛丼並盛, サラダ (780kcal, P28.0g)
  夜食: プロテイン, おにぎり (360kcal, P27.0g)
  その他: コーヒー×2 (10kcal)

📝 評価: タンパク質はほぼ達成。3食+間食で
分散摂取できており吸収効率も◎。カリウム源の
野菜・果物がやや少なめ。
```

### Interim Check (thread reply)

```
⏰ 15:00 チェック

ここまで: 870kcal / P: 68.0g
残り目標: 1,805kcal / P: 78.0g

💡 夕食でP30g以上摂れば、寝る前のプロテインで
目標達成ペースです。
```

### Final Check 0:00 (thread reply)

```
🌙 最終チェック

今日の合計: 2,150kcal / P: 128.0g
目標比: カロリー 80% / タンパク質 88%

⚡ プロテイン1杯(P24g)で目標達成です。
飲んだら「プロテイン」と投稿してください。
```

## Scope

### In scope (MVP)

- 3 new DB tables (nutrition_items, nutrition_logs, nutrition_targets)
- `[nutrition]` section in alt.toml
- `nutrition-check-cloud` skill (5 schedules)
- 6 preset nutrition_items (protein, EAA, 3 supplements + bundle)
- Target auto-calculation from body composition goals

### Out of scope

- Webapp nutrition dashboard
- Exercise tracking (personal trainer records, etc.)
- PFC / micronutrient detailed tracking
- Real-time processing (webhook)
