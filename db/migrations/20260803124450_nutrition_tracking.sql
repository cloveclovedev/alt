-- Create enum type "nutrition_entry_source"
CREATE TYPE "public"."nutrition_entry_source" AS ENUM ('manual', 'ai_photo', 'ai_text', 'catalog');
-- Create enum type "nutrition_catalog_kind"
CREATE TYPE "public"."nutrition_catalog_kind" AS ENUM ('food', 'supplement');
-- Create enum type "nutrition_meal_type"
CREATE TYPE "public"."nutrition_meal_type" AS ENUM ('breakfast', 'lunch', 'dinner', 'snack');
-- Create "nutrition_catalog" table
CREATE TABLE "public"."nutrition_catalog" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "name" text NOT NULL,
  "calories_kcal" integer NOT NULL,
  "protein_g" numeric(6,1) NOT NULL,
  "kind" "public"."nutrition_catalog_kind" NOT NULL DEFAULT 'food',
  "source" text NOT NULL DEFAULT 'manual',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "nutrition_catalog_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "nutrition_catalog_calories_non_negative" CHECK (calories_kcal >= 0),
  CONSTRAINT "nutrition_catalog_name_not_blank" CHECK (btrim(name) <> ''::text),
  CONSTRAINT "nutrition_catalog_protein_non_negative" CHECK (protein_g >= (0)::numeric)
);
-- Create index "nutrition_catalog_user_id_idx" to table: "nutrition_catalog"
CREATE INDEX "nutrition_catalog_user_id_idx" ON "public"."nutrition_catalog" ("user_id");
-- Create index "nutrition_catalog_user_name_key" to table: "nutrition_catalog"
CREATE UNIQUE INDEX "nutrition_catalog_user_name_key" ON "public"."nutrition_catalog" ("user_id", "name");
-- Create "nutrition_entries" table
CREATE TABLE "public"."nutrition_entries" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "logged_date" date NOT NULL,
  "meal_type" "public"."nutrition_meal_type" NOT NULL,
  "name" text NOT NULL,
  "calories_kcal" integer NOT NULL,
  "protein_g" numeric(6,1) NOT NULL,
  "source" "public"."nutrition_entry_source" NOT NULL,
  "catalog_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "nutrition_entries_catalog_id_fkey" FOREIGN KEY ("catalog_id") REFERENCES "public"."nutrition_catalog" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "nutrition_entries_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "nutrition_entries_calories_non_negative" CHECK (calories_kcal >= 0),
  CONSTRAINT "nutrition_entries_name_not_blank" CHECK (btrim(name) <> ''::text),
  CONSTRAINT "nutrition_entries_protein_non_negative" CHECK (protein_g >= (0)::numeric)
);
-- Create index "nutrition_entries_catalog_id_idx" to table: "nutrition_entries"
CREATE INDEX "nutrition_entries_catalog_id_idx" ON "public"."nutrition_entries" ("catalog_id");
-- Create index "nutrition_entries_user_logged_date_idx" to table: "nutrition_entries"
CREATE INDEX "nutrition_entries_user_logged_date_idx" ON "public"."nutrition_entries" ("user_id", "logged_date");
-- Create "nutrition_targets" table
CREATE TABLE "public"."nutrition_targets" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "effective_on" date NOT NULL,
  "calories_kcal" integer NOT NULL,
  "protein_g" numeric(6,1) NOT NULL,
  "rationale" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "nutrition_targets_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "nutrition_targets_calories_non_negative" CHECK (calories_kcal >= 0),
  CONSTRAINT "nutrition_targets_protein_non_negative" CHECK (protein_g >= (0)::numeric)
);
-- Create index "nutrition_targets_user_effective_on_key" to table: "nutrition_targets"
CREATE UNIQUE INDEX "nutrition_targets_user_effective_on_key" ON "public"."nutrition_targets" ("user_id", "effective_on");
