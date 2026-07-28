schema "public" {}

enum "user_status" {
  schema = schema.public
  values = ["active", "disabled"]
}

table "users" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "status" {
    type    = enum.user_status
    default = "active"
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
}

table "profiles" {
  schema = schema.public

  column "user_id" {
    type = uuid
    null = false
  }
  column "display_name" {
    type = text
    null = false
  }
  column "timezone" {
    type    = text
    default = "Asia/Tokyo"
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
    columns = [column.user_id]
  }

  foreign_key "profiles_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }

  check "profiles_display_name_not_blank" {
    expr = "btrim(display_name) <> ''"
  }
  check "profiles_timezone_not_blank" {
    expr = "btrim(timezone) <> ''"
  }
}
