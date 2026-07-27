-- Create "users" table
CREATE TABLE "public"."users" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "display_name" text NOT NULL,
  "timezone" text NOT NULL DEFAULT 'Asia/Tokyo',
  "status" text NOT NULL DEFAULT 'active',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "users_display_name_not_blank" CHECK (btrim(display_name) <> ''::text),
  CONSTRAINT "users_status_check" CHECK (status = ANY (ARRAY['active'::text, 'disabled'::text]))
);
-- Create "daily_plans" table
CREATE TABLE "public"."daily_plans" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "plan_date" date NOT NULL,
  "revision" integer NOT NULL,
  "status" text NOT NULL DEFAULT 'accepted',
  "summary" text NOT NULL,
  "context_snapshot" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "daily_plans_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "daily_plans_revision_positive" CHECK (revision > 0),
  CONSTRAINT "daily_plans_status_check" CHECK (status = ANY (ARRAY['draft'::text, 'accepted'::text, 'archived'::text])),
  CONSTRAINT "daily_plans_summary_not_blank" CHECK (btrim(summary) <> ''::text)
);
-- Create index "daily_plans_one_accepted_idx" to table: "daily_plans"
CREATE UNIQUE INDEX "daily_plans_one_accepted_idx" ON "public"."daily_plans" ("user_id", "plan_date") WHERE (status = 'accepted'::text);
-- Create index "daily_plans_user_date_revision_key" to table: "daily_plans"
CREATE UNIQUE INDEX "daily_plans_user_date_revision_key" ON "public"."daily_plans" ("user_id", "plan_date", "revision");
-- Create "journal_entries" table
CREATE TABLE "public"."journal_entries" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "kind" text NOT NULL DEFAULT 'note',
  "body" text NOT NULL,
  "occurred_at" timestamptz NOT NULL DEFAULT now(),
  "source" text NOT NULL DEFAULT 'web',
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "journal_entries_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "journal_entries_body_not_blank" CHECK (btrim(body) <> ''::text),
  CONSTRAINT "journal_entries_kind_check" CHECK (kind = ANY (ARRAY['note'::text, 'reflection'::text, 'idea'::text]))
);
-- Create index "journal_entries_user_occurred_at_idx" to table: "journal_entries"
CREATE INDEX "journal_entries_user_occurred_at_idx" ON "public"."journal_entries" ("user_id", "occurred_at");
-- Create "tasks" table
CREATE TABLE "public"."tasks" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "title" text NOT NULL,
  "notes" text NULL,
  "status" text NOT NULL DEFAULT 'active',
  "priority" text NULL,
  "due_date" date NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "tasks_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "tasks_priority_check" CHECK ((priority IS NULL) OR (priority = ANY (ARRAY['P0'::text, 'P1'::text, 'P2'::text, 'P3'::text]))),
  CONSTRAINT "tasks_status_check" CHECK (status = ANY (ARRAY['active'::text, 'backlog'::text, 'done'::text, 'cancelled'::text])),
  CONSTRAINT "tasks_title_not_blank" CHECK (btrim(title) <> ''::text)
);
-- Create index "tasks_user_active_due_idx" to table: "tasks"
CREATE INDEX "tasks_user_active_due_idx" ON "public"."tasks" ("user_id", "status", "due_date") WHERE (status = ANY (ARRAY['active'::text, 'backlog'::text]));
