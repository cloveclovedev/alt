table "body_measurements" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "measured_at" {
    type = timestamptz
    null = false
  }
  column "weight_kg" {
    type = decimal(5,2)
    null = false
  }
  column "skeletal_muscle_mass_kg" {
    type = decimal(5,2)
    null = true
  }
  column "muscle_mass_kg" {
    type = decimal(5,2)
    null = true
  }
  column "body_fat_mass_kg" {
    type = decimal(5,2)
    null = true
  }
  column "body_fat_percent" {
    type = decimal(4,1)
    null = true
  }
  column "bmi" {
    type = decimal(4,1)
    null = true
  }
  column "basal_metabolic_rate" {
    type = integer
    null = true
  }
  column "inbody_score" {
    type = decimal(4,1)
    null = true
  }
  column "waist_hip_ratio" {
    type = decimal(3,2)
    null = true
  }
  column "visceral_fat_level" {
    type = integer
    null = true
  }
  column "ffmi" {
    type = decimal(4,2)
    null = true
  }
  column "skeletal_muscle_ratio" {
    type = decimal(4,1)
    null = true
  }
  column "source" {
    type    = text
    null    = false
    default = "inbody_csv"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_body_measurements_measured_at" {
    columns = [column.measured_at]
    unique  = true
  }
}
