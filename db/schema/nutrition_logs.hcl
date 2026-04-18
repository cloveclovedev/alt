table "nutrition_logs" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "logged_date" {
    type = date
    null = false
  }
  column "meal_type" {
    type = text
    null = false
  }
  column "description" {
    type = text
    null = false
  }
  column "calories_kcal" {
    type = decimal(6,1)
    null = true
  }
  column "protein_g" {
    type = decimal(5,1)
    null = true
  }
  column "supplement_taken" {
    type    = boolean
    null    = false
    default = false
  }
  column "source_message_id" {
    type = text
    null = true
  }
  column "estimated_by" {
    type = text
    null = false
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

  index "idx_nutrition_logs_date" {
    columns = [column.logged_date]
  }

  index "idx_nutrition_logs_source_message" {
    columns = [column.source_message_id]
    unique  = true
    where   = "source_message_id IS NOT NULL"
  }
}
