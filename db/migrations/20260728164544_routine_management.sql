-- Create enum type "routine_status"
CREATE TYPE "public"."routine_status" AS ENUM ('active', 'inactive');
-- Create enum type "routine_event_kind"
CREATE TYPE "public"."routine_event_kind" AS ENUM ('baseline', 'completed');
-- Create "routine_categories" table
CREATE TABLE "public"."routine_categories" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "name" text NOT NULL,
  "position" integer NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "routine_categories_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "routine_categories_name_not_blank" CHECK (btrim(name) <> ''::text)
);
-- Create index "routine_categories_user_name_key" to table: "routine_categories"
CREATE UNIQUE INDEX "routine_categories_user_name_key" ON "public"."routine_categories" ("user_id", (btrim(name)));
-- Create index "routine_categories_user_position_idx" to table: "routine_categories"
CREATE INDEX "routine_categories_user_position_idx" ON "public"."routine_categories" ("user_id", "position");
-- Create "routines" table
CREATE TABLE "public"."routines" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "category_id" uuid NOT NULL,
  "name" text NOT NULL,
  "notes" text NOT NULL DEFAULT '',
  "status" "public"."routine_status" NOT NULL DEFAULT 'active',
  "interval_days" integer NOT NULL,
  "available_weekdays" smallint[] NOT NULL DEFAULT '{}',
  "active_months" smallint[] NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "routines_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "public"."routine_categories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "routines_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "routines_active_months_valid" CHECK ((active_months <@ ARRAY[(1)::smallint, (2)::smallint, (3)::smallint, (4)::smallint, (5)::smallint, (6)::smallint, (7)::smallint, (8)::smallint, (9)::smallint, (10)::smallint, (11)::smallint, (12)::smallint])),
  CONSTRAINT "routines_available_weekdays_valid" CHECK ((available_weekdays <@ ARRAY[(1)::smallint, (2)::smallint, (3)::smallint, (4)::smallint, (5)::smallint, (6)::smallint, (7)::smallint])),
  CONSTRAINT "routines_interval_days_positive" CHECK (interval_days > 0),
  CONSTRAINT "routines_name_not_blank" CHECK (btrim(name) <> ''::text)
);
-- Create index "routines_category_id_idx" to table: "routines"
CREATE INDEX "routines_category_id_idx" ON "public"."routines" ("category_id");
-- Create index "routines_user_name_key" to table: "routines"
CREATE UNIQUE INDEX "routines_user_name_key" ON "public"."routines" ("user_id", "name");
-- Create index "routines_user_status_idx" to table: "routines"
CREATE INDEX "routines_user_status_idx" ON "public"."routines" ("user_id", "status");
-- Create "routine_events" table
CREATE TABLE "public"."routine_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "routine_id" uuid NOT NULL,
  "kind" "public"."routine_event_kind" NOT NULL,
  "completed_on" date NOT NULL,
  "note" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "routine_events_routine_id_fkey" FOREIGN KEY ("routine_id") REFERENCES "public"."routines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "routine_events_routine_anchor_idx" to table: "routine_events"
CREATE INDEX "routine_events_routine_anchor_idx" ON "public"."routine_events" ("routine_id", "completed_on", "created_at", "id");
-- Create index "routine_events_routine_id_idx" to table: "routine_events"
CREATE INDEX "routine_events_routine_id_idx" ON "public"."routine_events" ("routine_id");
