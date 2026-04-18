schema "public" {}

table "entries" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "type" {
    type = text
    null = false
  }
  column "title" {
    type = text
    null = false
  }
  column "content" {
    type = text
    null = true
  }
  column "status" {
    type = text
    null = true
  }
  column "tags" {
    type    = jsonb
    default = "[]"
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

  index "idx_entries_type" {
    columns = [column.type]
  }
  index "idx_entries_tags" {
    type    = GIN
    columns = [column.tags]
  }
  index "idx_entries_created" {
    columns = [column.created_at]
  }
}
