-- Create "body_measurement_goals" table
CREATE TABLE "body_measurement_goals" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "metric" text NOT NULL,
  "target_value" numeric(6,2) NOT NULL,
  "start_value" numeric(6,2) NULL,
  "start_date" date NOT NULL,
  "target_date" date NOT NULL,
  "status" text NOT NULL DEFAULT 'active',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create index "idx_body_measurement_goals_active_metric" to table: "body_measurement_goals"
CREATE UNIQUE INDEX "idx_body_measurement_goals_active_metric" ON "body_measurement_goals" ("metric") WHERE (status = 'active'::text);
