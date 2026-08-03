# Shared AI foundation (internal/ai): purpose-based model assignment and
# non-content usage metadata. These tables are feature-neutral so any feature can
# request inference through internal/ai without a feature-to-feature dependency.

table "ai_model_assignments" {
  schema = schema.public

  column "user_id" {
    type = uuid
    null = false
  }
  column "purpose" {
    type = text
    null = false
  }
  column "model_id" {
    type = text
    null = false
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
    columns = [column.user_id, column.purpose]
  }
  foreign_key "ai_model_assignments_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  check "ai_model_assignments_purpose_not_blank" {
    expr = "btrim(purpose) <> ''"
  }
  check "ai_model_assignments_model_id_not_blank" {
    expr = "btrim(model_id) <> ''"
  }
}

table "ai_generations" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "purpose" {
    type = text
    null = false
  }
  column "model_id" {
    type = text
    null = false
  }
  column "provider_generation_id" {
    type    = text
    null    = false
    default = ""
  }
  column "prompt_version" {
    type = text
    null = false
  }
  column "prompt_tokens" {
    type    = integer
    null    = false
    default = 0
  }
  column "completion_tokens" {
    type    = integer
    null    = false
    default = 0
  }
  column "status" {
    type = text
    null = false
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
  foreign_key "ai_generations_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  check "ai_generations_purpose_not_blank" {
    expr = "btrim(purpose) <> ''"
  }
  check "ai_generations_model_id_not_blank" {
    expr = "btrim(model_id) <> ''"
  }
  check "ai_generations_prompt_version_not_blank" {
    expr = "btrim(prompt_version) <> ''"
  }
  check "ai_generations_status_not_blank" {
    expr = "btrim(status) <> ''"
  }
  check "ai_generations_token_counts_non_negative" {
    expr = "prompt_tokens >= 0 AND completion_tokens >= 0"
  }
  index "ai_generations_user_id_created_at_idx" {
    columns = [column.user_id, column.created_at]
  }
}
