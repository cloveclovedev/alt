table "nutrition_targets" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "calories_kcal" {
    type = decimal(6,1)
    null = false
  }
  column "protein_g" {
    type = decimal(5,1)
    null = false
  }
  column "effective_from" {
    type = date
    null = false
  }
  column "effective_until" {
    type = date
    null = true
  }
  column "rationale" {
    type = text
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

  index "idx_nutrition_targets_effective" {
    columns = [column.effective_from, column.effective_until]
  }
}
