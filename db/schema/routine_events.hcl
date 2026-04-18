table "routine_events" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "routine_name" {
    type = text
    null = false
  }
  column "category" {
    type = text
    null = false
  }
  column "completed_at" {
    type    = timestamptz
    null    = false
    default = sql("now()")
  }
  column "kind" {
    type    = text
    null    = false
    default = "completed"
  }
  column "note" {
    type = text
    null = true
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_routine_events_name" {
    columns = [column.routine_name]
  }
}
