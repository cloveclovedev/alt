table "journal_entries" {
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
    type    = text
    default = "note"
  }
  column "body" {
    type = text
    null = false
  }
  column "occurred_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "source" {
    type    = text
    default = "web"
  }
  column "metadata" {
    type    = jsonb
    default = "{}"
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

  foreign_key "journal_entries_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }

  check "journal_entries_kind_check" {
    expr = "kind IN ('note', 'reflection', 'idea')"
  }
  check "journal_entries_body_not_blank" {
    expr = "btrim(body) <> ''"
  }

  index "journal_entries_user_occurred_at_idx" {
    columns = [column.user_id, column.occurred_at]
  }
}

table "tasks" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "title" {
    type = text
    null = false
  }
  column "notes" {
    type = text
    null = true
  }
  column "status" {
    type    = text
    default = "active"
  }
  column "priority" {
    type = text
    null = true
  }
  column "due_date" {
    type = date
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

  foreign_key "tasks_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }

  check "tasks_title_not_blank" {
    expr = "btrim(title) <> ''"
  }
  check "tasks_status_check" {
    expr = "status IN ('active', 'backlog', 'done', 'cancelled')"
  }
  check "tasks_priority_check" {
    expr = "priority IS NULL OR priority IN ('P0', 'P1', 'P2', 'P3')"
  }

  index "tasks_user_active_due_idx" {
    columns = [column.user_id, column.status, column.due_date]
    where   = "status IN ('active', 'backlog')"
  }
}

table "daily_plans" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "plan_date" {
    type = date
    null = false
  }
  column "revision" {
    type = integer
    null = false
  }
  column "status" {
    type    = text
    default = "accepted"
  }
  column "summary" {
    type = text
    null = false
  }
  column "context_snapshot" {
    type    = jsonb
    default = "{}"
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

  foreign_key "daily_plans_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }

  check "daily_plans_revision_positive" {
    expr = "revision > 0"
  }
  check "daily_plans_status_check" {
    expr = "status IN ('draft', 'accepted', 'archived')"
  }
  check "daily_plans_summary_not_blank" {
    expr = "btrim(summary) <> ''"
  }

  index "daily_plans_user_date_revision_key" {
    unique  = true
    columns = [column.user_id, column.plan_date, column.revision]
  }
  index "daily_plans_one_accepted_idx" {
    unique  = true
    columns = [column.user_id, column.plan_date]
    where   = "status = 'accepted'"
  }
}
