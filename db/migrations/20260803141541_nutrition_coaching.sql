-- Create "nutrition_coaching" table
CREATE TABLE "public"."nutrition_coaching" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "coached_date" date NOT NULL,
  "evaluation_markdown" text NOT NULL,
  "suggestion_markdown" text NOT NULL,
  "model_id" text NOT NULL,
  "prompt_version" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "nutrition_coaching_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "nutrition_coaching_user_coached_date_key" to table: "nutrition_coaching"
CREATE UNIQUE INDEX "nutrition_coaching_user_coached_date_key" ON "public"."nutrition_coaching" ("user_id", "coached_date");
