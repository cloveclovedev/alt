-- Create "body_measurements" table
CREATE TABLE "body_measurements" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "measured_at" timestamptz NOT NULL,
  "weight_kg" numeric(5,2) NOT NULL,
  "skeletal_muscle_mass_kg" numeric(5,2) NULL,
  "muscle_mass_kg" numeric(5,2) NULL,
  "body_fat_mass_kg" numeric(5,2) NULL,
  "body_fat_percent" numeric(4,1) NULL,
  "bmi" numeric(4,1) NULL,
  "basal_metabolic_rate" integer NULL,
  "inbody_score" numeric(4,1) NULL,
  "waist_hip_ratio" numeric(3,2) NULL,
  "visceral_fat_level" integer NULL,
  "ffmi" numeric(4,2) NULL,
  "skeletal_muscle_ratio" numeric(4,1) NULL,
  "source" text NOT NULL DEFAULT 'inbody_csv',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create index "idx_body_measurements_measured_at" to table: "body_measurements"
CREATE UNIQUE INDEX "idx_body_measurements_measured_at" ON "body_measurements" ("measured_at");
-- Create "entries" table
CREATE TABLE "entries" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "type" text NOT NULL,
  "title" text NOT NULL,
  "content" text NULL,
  "status" text NULL,
  "tags" jsonb NOT NULL DEFAULT '[]',
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create index "idx_entries_created" to table: "entries"
CREATE INDEX "idx_entries_created" ON "entries" ("created_at");
-- Create index "idx_entries_tags" to table: "entries"
CREATE INDEX "idx_entries_tags" ON "entries" USING GIN ("tags");
-- Create index "idx_entries_type" to table: "entries"
CREATE INDEX "idx_entries_type" ON "entries" ("type");
-- Create "routine_events" table
CREATE TABLE "routine_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "routine_name" text NOT NULL,
  "category" text NOT NULL,
  "completed_at" timestamptz NOT NULL DEFAULT now(),
  "kind" text NOT NULL DEFAULT 'completed',
  "note" text NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_routine_events_name" to table: "routine_events"
CREATE INDEX "idx_routine_events_name" ON "routine_events" ("routine_name");
