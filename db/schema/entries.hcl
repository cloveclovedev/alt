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
  column "metadata" {
    type    = jsonb
    default = "{}"
  }
  column "parent_id" {
    type = uuid
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

  foreign_key "fk_entries_parent" {
    columns     = [column.parent_id]
    ref_columns = [column.id]
    on_delete   = SET_NULL
  }

  index "idx_entries_type" {
    columns = [column.type]
  }
  index "idx_entries_created" {
    columns = [column.created_at]
  }
  index "idx_entries_parent" {
    columns = [column.parent_id]
    where   = "parent_id IS NOT NULL"
  }
}
