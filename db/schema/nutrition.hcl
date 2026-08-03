# Nutrition tracking (internal/nutrition): a reusable catalog, a mutable intake
# log, and effective-dated targets. Metrics are calories and protein only for the
# MVP. See docs/designs/nutrition-tracking.md.

enum "nutrition_catalog_kind" {
  schema = schema.public
  values = ["food", "supplement"]
}

enum "nutrition_meal_type" {
  schema = schema.public
  values = ["breakfast", "lunch", "dinner", "snack"]
}

enum "nutrition_entry_source" {
  schema = schema.public
  values = ["manual", "ai_photo", "ai_text", "catalog"]
}

# Pre-registered reusable items for quick entry and AI name matching. Calories and
# protein are per the registered serving.
table "nutrition_catalog" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "calories_kcal" {
    type = integer
    null = false
  }
  column "protein_g" {
    type = sql("numeric(6,1)")
    null = false
  }
  column "kind" {
    type    = enum.nutrition_catalog_kind
    null    = false
    default = "food"
  }
  column "source" {
    type    = text
    null    = false
    default = "manual"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "nutrition_catalog_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  check "nutrition_catalog_name_not_blank" {
    expr = "btrim(name) <> ''"
  }
  check "nutrition_catalog_calories_non_negative" {
    expr = "calories_kcal >= 0"
  }
  check "nutrition_catalog_protein_non_negative" {
    expr = "protein_g >= 0"
  }
  index "nutrition_catalog_user_name_key" {
    unique  = true
    columns = [column.user_id, column.name]
  }
  index "nutrition_catalog_user_id_idx" {
    columns = [column.user_id]
  }
}

# A mutable intake log, one row per consumed item. A "meal" is simply the rows
# sharing (logged_date, meal_type). name/calories/protein are denormalized
# snapshots so editing or deleting a catalog item never rewrites history.
table "nutrition_entries" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "logged_date" {
    type = date
    null = false
  }
  column "meal_type" {
    type = enum.nutrition_meal_type
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "calories_kcal" {
    type = integer
    null = false
  }
  column "protein_g" {
    type = sql("numeric(6,1)")
    null = false
  }
  column "source" {
    type = enum.nutrition_entry_source
    null = false
  }
  column "catalog_id" {
    type = uuid
    null = true
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "nutrition_entries_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  foreign_key "nutrition_entries_catalog_id_fkey" {
    columns     = [column.catalog_id]
    ref_columns = [table.nutrition_catalog.column.id]
    on_delete   = SET_NULL
  }
  check "nutrition_entries_name_not_blank" {
    expr = "btrim(name) <> ''"
  }
  check "nutrition_entries_calories_non_negative" {
    expr = "calories_kcal >= 0"
  }
  check "nutrition_entries_protein_non_negative" {
    expr = "protein_g >= 0"
  }
  index "nutrition_entries_user_logged_date_idx" {
    columns = [column.user_id, column.logged_date]
  }
  index "nutrition_entries_catalog_id_idx" {
    columns = [column.catalog_id]
  }
}

# Effective-dated daily targets. The applicable target for a date D is the row
# with the greatest effective_on <= D; a newer effective_on implicitly supersedes
# older ones, so there is no end column and no overlap bookkeeping.
table "nutrition_targets" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "effective_on" {
    type = date
    null = false
  }
  column "calories_kcal" {
    type = integer
    null = false
  }
  column "protein_g" {
    type = sql("numeric(6,1)")
    null = false
  }
  column "rationale" {
    type = text
    null = true
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "nutrition_targets_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  check "nutrition_targets_calories_non_negative" {
    expr = "calories_kcal >= 0"
  }
  check "nutrition_targets_protein_non_negative" {
    expr = "protein_g >= 0"
  }
  index "nutrition_targets_user_effective_on_key" {
    unique  = true
    columns = [column.user_id, column.effective_on]
  }
}
