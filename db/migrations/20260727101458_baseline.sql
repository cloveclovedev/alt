-- Create enum type "user_status"
CREATE TYPE "public"."user_status" AS ENUM ('active', 'disabled');
-- Create enum type "plan_kind"
CREATE TYPE "public"."plan_kind" AS ENUM ('daily', 'weekly');
-- Create "users" table
CREATE TABLE "public"."users" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "status" "public"."user_status" NOT NULL DEFAULT 'active',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create "plans" table
CREATE TABLE "public"."plans" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "kind" "public"."plan_kind" NOT NULL,
  "period_start" date NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "plans_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "plans_weekly_period_start_monday" CHECK ((kind <> 'weekly'::public.plan_kind) OR (EXTRACT(isodow FROM period_start) = (1)::numeric))
);
-- Create index "plans_user_kind_period_start_key" to table: "plans"
CREATE UNIQUE INDEX "plans_user_kind_period_start_key" ON "public"."plans" ("user_id", "kind", "period_start");
-- Create "plan_revisions" table
CREATE TABLE "public"."plan_revisions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "plan_id" uuid NOT NULL,
  "revision" integer NOT NULL,
  "summary_markdown" text NOT NULL,
  "content_markdown" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_revisions_plan_id_fkey" FOREIGN KEY ("plan_id") REFERENCES "public"."plans" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "plan_revisions_content_markdown_not_blank" CHECK (btrim(content_markdown) <> ''::text),
  CONSTRAINT "plan_revisions_revision_positive" CHECK (revision > 0),
  CONSTRAINT "plan_revisions_summary_markdown_not_blank" CHECK (btrim(summary_markdown) <> ''::text)
);
-- Create index "plan_revisions_plan_revision_key" to table: "plan_revisions"
CREATE UNIQUE INDEX "plan_revisions_plan_revision_key" ON "public"."plan_revisions" ("plan_id", "revision");
-- Create "profiles" table
CREATE TABLE "public"."profiles" (
  "user_id" uuid NOT NULL,
  "display_name" text NOT NULL,
  "timezone" text NOT NULL DEFAULT 'Asia/Tokyo',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("user_id"),
  CONSTRAINT "profiles_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "profiles_display_name_not_blank" CHECK (btrim(display_name) <> ''::text),
  CONSTRAINT "profiles_timezone_not_blank" CHECK (btrim(timezone) <> ''::text)
);
