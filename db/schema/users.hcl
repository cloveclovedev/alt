table "users" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "display_name" {
    type = text
    null = false
  }
  column "timezone" {
    type    = text
    default = "Asia/Tokyo"
  }
  column "status" {
    type    = text
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

  check "users_display_name_not_blank" {
    expr = "btrim(display_name) <> ''"
  }
  check "users_status_check" {
    expr = "status IN ('active', 'disabled')"
  }
}
