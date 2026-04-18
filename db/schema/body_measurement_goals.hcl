table "body_measurement_goals" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "metric" {
    type = text
    null = false
  }
  column "target_value" {
    type = decimal(6,2)
    null = false
  }
  column "start_value" {
    type = decimal(6,2)
    null = true
  }
  column "start_date" {
    type = date
    null = false
  }
  column "target_date" {
    type = date
    null = false
  }
  column "status" {
    type    = text
    null    = false
    default = "active"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_body_measurement_goals_active_metric" {
    columns = [column.metric]
    unique  = true
    where   = "status = 'active'"
  }
}
