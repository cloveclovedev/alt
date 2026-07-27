table "config" {
  schema = schema.public

  column "key" {
    type = text
    null = false
  }
  column "value" {
    type = jsonb
    null = false
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
    columns = [column.key]
  }
}
