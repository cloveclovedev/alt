enum "routine_status" {
  schema = schema.public
  values = ["active", "inactive"]
}

enum "routine_event_kind" {
  schema = schema.public
  values = ["baseline", "completed"]
}

table "routine_categories" {
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
  column "position" {
    type = integer
    null = false
  }
  column "active" {
    type    = boolean
    default = true
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

  foreign_key "routine_categories_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }

  check "routine_categories_name_not_blank" {
    expr = "btrim(name) <> ''"
  }

  index "routine_categories_user_name_key" {
    unique = true
    on {
      column = column.user_id
    }
    on {
      expr = "btrim(name)"
    }
  }
  index "routine_categories_user_position_idx" {
    columns = [column.user_id, column.position]
  }
}

table "routines" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "category_id" {
    type = uuid
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "notes" {
    type    = text
    default = ""
  }
  column "status" {
    type    = enum.routine_status
    default = "active"
  }
  column "interval_days" {
    type = integer
    null = false
  }
  column "available_weekdays" {
    type    = sql("smallint[]")
    default = sql("'{}'::smallint[]")
  }
  column "active_months" {
    type    = sql("smallint[]")
    default = sql("'{}'::smallint[]")
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

  foreign_key "routines_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  foreign_key "routines_category_id_fkey" {
    columns     = [column.category_id]
    ref_columns = [table.routine_categories.column.id]
  }

  check "routines_name_not_blank" {
    expr = "btrim(name) <> ''"
  }
  check "routines_interval_days_positive" {
    expr = "interval_days > 0"
  }
  check "routines_available_weekdays_valid" {
    expr = "available_weekdays <@ ARRAY[1, 2, 3, 4, 5, 6, 7]::smallint[]"
  }
  check "routines_active_months_valid" {
    expr = "active_months <@ ARRAY[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]::smallint[]"
  }

  index "routines_user_name_key" {
    unique  = true
    columns = [column.user_id, column.name]
  }
  index "routines_user_status_idx" {
    columns = [column.user_id, column.status]
  }
  index "routines_category_id_idx" {
    columns = [column.category_id]
  }
}

table "routine_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "routine_id" {
    type = uuid
    null = false
  }
  column "kind" {
    type = enum.routine_event_kind
    null = false
  }
  column "completed_on" {
    type = date
    null = false
  }
  column "note" {
    type    = text
    default = ""
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

  foreign_key "routine_events_routine_id_fkey" {
    columns     = [column.routine_id]
    ref_columns = [table.routines.column.id]
    on_delete   = CASCADE
  }

  index "routine_events_routine_id_idx" {
    columns = [column.routine_id]
  }
  index "routine_events_routine_anchor_idx" {
    columns = [column.routine_id, column.completed_on, column.created_at, column.id]
  }
}
