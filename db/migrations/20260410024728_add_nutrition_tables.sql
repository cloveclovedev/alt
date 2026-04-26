-- Create "nutrition_items" table
CREATE TABLE "nutrition_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "calories_kcal" numeric(6,1) NOT NULL,
  "protein_g" numeric(5,1) NOT NULL,
  "source" text NOT NULL DEFAULT 'user_registered',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create index "idx_nutrition_items_name" to table: "nutrition_items"
CREATE UNIQUE INDEX "idx_nutrition_items_name" ON "nutrition_items" ("name");
-- Create "nutrition_logs" table
CREATE TABLE "nutrition_logs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "logged_date" date NOT NULL,
  "meal_type" text NOT NULL,
  "description" text NOT NULL,
  "calories_kcal" numeric(6,1) NULL,
  "protein_g" numeric(5,1) NULL,
  "supplement_taken" boolean NOT NULL DEFAULT false,
  "source_message_id" text NULL,
  "estimated_by" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create index "idx_nutrition_logs_date" to table: "nutrition_logs"
CREATE INDEX "idx_nutrition_logs_date" ON "nutrition_logs" ("logged_date");
-- Create index "idx_nutrition_logs_source_message" to table: "nutrition_logs"
CREATE UNIQUE INDEX "idx_nutrition_logs_source_message" ON "nutrition_logs" ("source_message_id") WHERE (source_message_id IS NOT NULL);
-- Create "nutrition_targets" table
CREATE TABLE "nutrition_targets" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "calories_kcal" numeric(6,1) NOT NULL,
  "protein_g" numeric(5,1) NOT NULL,
  "effective_from" date NOT NULL,
  "effective_until" date NULL,
  "rationale" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create index "idx_nutrition_targets_effective" to table: "nutrition_targets"
CREATE INDEX "idx_nutrition_targets_effective" ON "nutrition_targets" ("effective_from", "effective_until");
