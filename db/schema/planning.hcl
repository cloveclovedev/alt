enum "plan_kind" {
  schema = schema.public
  values = ["daily", "weekly"]
}

table "plans" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "kind" {
    type = enum.plan_kind
    null = false
  }
  column "period_start" {
    type = date
    null = false
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "plans_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }

  check "plans_weekly_period_start_monday" {
    expr = "kind <> 'weekly' OR EXTRACT(ISODOW FROM period_start) = 1"
  }

  index "plans_user_kind_period_start_key" {
    unique  = true
    columns = [column.user_id, column.kind, column.period_start]
  }
}

table "plan_revisions" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "plan_id" {
    type = uuid
    null = false
  }
  column "revision" {
    type = integer
    null = false
  }
  column "summary_markdown" {
    type = text
    null = false
  }
  column "content_markdown" {
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

  foreign_key "plan_revisions_plan_id_fkey" {
    columns     = [column.plan_id]
    ref_columns = [table.plans.column.id]
    on_delete   = CASCADE
  }

  check "plan_revisions_revision_positive" {
    expr = "revision > 0"
  }
  check "plan_revisions_summary_markdown_not_blank" {
    expr = "btrim(summary_markdown) <> ''"
  }
  check "plan_revisions_content_markdown_not_blank" {
    expr = "btrim(content_markdown) <> ''"
  }

  index "plan_revisions_plan_revision_key" {
    unique  = true
    columns = [column.plan_id, column.revision]
  }
}
