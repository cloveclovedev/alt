table "nutrition_items" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "name" {
    type = text
    null = false
  }
  column "calories_kcal" {
    type = decimal(6,1)
    null = false
  }
  column "protein_g" {
    type = decimal(5,1)
    null = false
  }
  column "source" {
    type    = text
    null    = false
    default = "user_registered"
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

  index "idx_nutrition_items_name" {
    columns = [column.name]
    unique  = true
  }
}
