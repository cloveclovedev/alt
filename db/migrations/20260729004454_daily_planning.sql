-- Create enum type "calendar_source_role"
CREATE TYPE "public"."calendar_source_role" AS ENUM ('commitment', 'optional', 'context');
-- Create enum type "daily_planning_session_status"
CREATE TYPE "public"."daily_planning_session_status" AS ENUM ('gathering', 'context_blocked', 'chatting', 'reviewing', 'finalized', 'cancelled');
-- Modify "plan_revisions" table
ALTER TABLE "public"."plan_revisions" ADD COLUMN "notes_markdown" text NULL;
-- Create enum type "daily_planning_message_role"
CREATE TYPE "public"."daily_planning_message_role" AS ENUM ('user', 'assistant', 'system');
-- Create "daily_planning_sessions" table
CREATE TABLE "public"."daily_planning_sessions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "plan_date" date NOT NULL,
  "status" "public"."daily_planning_session_status" NOT NULL,
  "preview_json" jsonb NULL,
  "confirmed_plan_revision_id" uuid NULL,
  "model_id" text NOT NULL DEFAULT '',
  "prompt_version" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  "finalized_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "daily_planning_sessions_confirmed_plan_revision_id_fkey" FOREIGN KEY ("confirmed_plan_revision_id") REFERENCES "public"."plan_revisions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "daily_planning_sessions_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "daily_planning_sessions_user_plan_date_active_key" to table: "daily_planning_sessions"
CREATE UNIQUE INDEX "daily_planning_sessions_user_plan_date_active_key" ON "public"."daily_planning_sessions" ("user_id", "plan_date") WHERE (status = ANY (ARRAY['gathering'::public.daily_planning_session_status, 'context_blocked'::public.daily_planning_session_status, 'chatting'::public.daily_planning_session_status, 'reviewing'::public.daily_planning_session_status]));
-- Create index "daily_planning_sessions_user_plan_date_idx" to table: "daily_planning_sessions"
CREATE INDEX "daily_planning_sessions_user_plan_date_idx" ON "public"."daily_planning_sessions" ("user_id", "plan_date");
-- Create "ai_generations" table
CREATE TABLE "public"."ai_generations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "session_id" uuid NOT NULL,
  "purpose" text NOT NULL,
  "model_id" text NOT NULL,
  "provider_generation_id" text NOT NULL DEFAULT '',
  "prompt_version" text NOT NULL,
  "prompt_tokens" integer NOT NULL DEFAULT 0,
  "completion_tokens" integer NOT NULL DEFAULT 0,
  "status" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "ai_generations_session_id_fkey" FOREIGN KEY ("session_id") REFERENCES "public"."daily_planning_sessions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "ai_generations_model_id_not_blank" CHECK (btrim(model_id) <> ''::text),
  CONSTRAINT "ai_generations_prompt_version_not_blank" CHECK (btrim(prompt_version) <> ''::text),
  CONSTRAINT "ai_generations_purpose_not_blank" CHECK (btrim(purpose) <> ''::text),
  CONSTRAINT "ai_generations_status_not_blank" CHECK (btrim(status) <> ''::text),
  CONSTRAINT "ai_generations_token_counts_non_negative" CHECK ((prompt_tokens >= 0) AND (completion_tokens >= 0))
);
-- Create index "ai_generations_session_id_idx" to table: "ai_generations"
CREATE INDEX "ai_generations_session_id_idx" ON "public"."ai_generations" ("session_id");
-- Create "ai_model_assignments" table
CREATE TABLE "public"."ai_model_assignments" (
  "user_id" uuid NOT NULL,
  "purpose" text NOT NULL,
  "model_id" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("user_id", "purpose"),
  CONSTRAINT "ai_model_assignments_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "ai_model_assignments_model_id_not_blank" CHECK (btrim(model_id) <> ''::text),
  CONSTRAINT "ai_model_assignments_purpose_not_blank" CHECK (btrim(purpose) <> ''::text)
);
-- Create "google_calendar_connections" table
CREATE TABLE "public"."google_calendar_connections" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "encrypted_refresh_token" bytea NOT NULL,
  "granted_scopes" text[] NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "google_calendar_connections_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "google_calendar_connections_user_id_key" to table: "google_calendar_connections"
CREATE UNIQUE INDEX "google_calendar_connections_user_id_key" ON "public"."google_calendar_connections" ("user_id");
-- Create "calendar_sources" table
CREATE TABLE "public"."calendar_sources" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "connection_id" uuid NOT NULL,
  "external_calendar_id" text NOT NULL,
  "display_name" text NOT NULL,
  "enabled" boolean NOT NULL DEFAULT false,
  "role" "public"."calendar_source_role" NOT NULL DEFAULT 'commitment',
  "planning_instructions" text NOT NULL DEFAULT '',
  "timezone" text NOT NULL DEFAULT 'UTC',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "calendar_sources_connection_id_fkey" FOREIGN KEY ("connection_id") REFERENCES "public"."google_calendar_connections" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "calendar_sources_display_name_not_blank" CHECK (btrim(display_name) <> ''::text),
  CONSTRAINT "calendar_sources_external_id_not_blank" CHECK (btrim(external_calendar_id) <> ''::text),
  CONSTRAINT "calendar_sources_timezone_not_blank" CHECK (btrim(timezone) <> ''::text)
);
-- Create index "calendar_sources_connection_enabled_idx" to table: "calendar_sources"
CREATE INDEX "calendar_sources_connection_enabled_idx" ON "public"."calendar_sources" ("connection_id", "enabled");
-- Create index "calendar_sources_connection_external_id_key" to table: "calendar_sources"
CREATE UNIQUE INDEX "calendar_sources_connection_external_id_key" ON "public"."calendar_sources" ("connection_id", "external_calendar_id");
-- Create "daily_planning_contexts" table
CREATE TABLE "public"."daily_planning_contexts" (
  "session_id" uuid NOT NULL,
  "calendar_context" jsonb NOT NULL DEFAULT '{}',
  "github_context" jsonb NOT NULL DEFAULT '{}',
  "routine_context" jsonb NOT NULL DEFAULT '{}',
  "calendar_status" text NOT NULL,
  "github_status" text NOT NULL,
  "routine_status" text NOT NULL,
  "gathered_at" timestamptz NOT NULL,
  PRIMARY KEY ("session_id"),
  CONSTRAINT "daily_planning_contexts_session_id_fkey" FOREIGN KEY ("session_id") REFERENCES "public"."daily_planning_sessions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "daily_planning_messages" table
CREATE TABLE "public"."daily_planning_messages" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "session_id" uuid NOT NULL,
  "sequence" integer NOT NULL,
  "role" "public"."daily_planning_message_role" NOT NULL,
  "content" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "daily_planning_messages_session_id_fkey" FOREIGN KEY ("session_id") REFERENCES "public"."daily_planning_sessions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "daily_planning_messages_content_not_blank" CHECK (btrim(content) <> ''::text),
  CONSTRAINT "daily_planning_messages_sequence_positive" CHECK (sequence > 0)
);
-- Create index "daily_planning_messages_session_sequence_key" to table: "daily_planning_messages"
CREATE UNIQUE INDEX "daily_planning_messages_session_sequence_key" ON "public"."daily_planning_messages" ("session_id", "sequence");
-- Create "github_repositories" table
CREATE TABLE "public"."github_repositories" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "owner" text NOT NULL,
  "name" text NOT NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  "planning_instructions" text NOT NULL DEFAULT '',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "github_repositories_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "github_repositories_name_not_blank" CHECK (btrim(name) <> ''::text),
  CONSTRAINT "github_repositories_owner_not_blank" CHECK (btrim(owner) <> ''::text)
);
-- Create index "github_repositories_user_enabled_idx" to table: "github_repositories"
CREATE INDEX "github_repositories_user_enabled_idx" ON "public"."github_repositories" ("user_id", "enabled");
-- Create index "github_repositories_user_owner_name_key" to table: "github_repositories"
CREATE UNIQUE INDEX "github_repositories_user_owner_name_key" ON "public"."github_repositories" ("user_id", "owner", "name");
-- Create "google_oauth_states" table
CREATE TABLE "public"."google_oauth_states" (
  "state_hash" bytea NOT NULL,
  "user_id" uuid NOT NULL,
  "encrypted_code_verifier" bytea NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("state_hash"),
  CONSTRAINT "google_oauth_states_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "google_oauth_states_expires_at_idx" to table: "google_oauth_states"
CREATE INDEX "google_oauth_states_expires_at_idx" ON "public"."google_oauth_states" ("expires_at");
-- Create "plan_revision_action_items" table
CREATE TABLE "public"."plan_revision_action_items" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "plan_revision_id" uuid NOT NULL,
  "title" text NOT NULL,
  "note" text NOT NULL DEFAULT '',
  "position" integer NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_revision_action_items_plan_revision_id_fkey" FOREIGN KEY ("plan_revision_id") REFERENCES "public"."plan_revisions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "plan_revision_action_items_position_non_negative" CHECK ("position" >= 0),
  CONSTRAINT "plan_revision_action_items_title_not_blank" CHECK (btrim(title) <> ''::text)
);
-- Create index "plan_revision_action_items_plan_revision_position_key" to table: "plan_revision_action_items"
CREATE UNIQUE INDEX "plan_revision_action_items_plan_revision_position_key" ON "public"."plan_revision_action_items" ("plan_revision_id", "position");
-- Create "plan_revision_calendar_events" table
CREATE TABLE "public"."plan_revision_calendar_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "plan_revision_id" uuid NOT NULL,
  "calendar_source_id" uuid NULL,
  "external_event_id" text NOT NULL,
  "title" text NOT NULL,
  "starts_at" timestamptz NULL,
  "ends_at" timestamptz NULL,
  "start_date" date NULL,
  "end_date" date NULL,
  "all_day" boolean NOT NULL,
  "role" "public"."calendar_source_role" NOT NULL,
  "html_url" text NOT NULL DEFAULT '',
  "position" integer NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_revision_calendar_events_calendar_source_id_fkey" FOREIGN KEY ("calendar_source_id") REFERENCES "public"."calendar_sources" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "plan_revision_calendar_events_plan_revision_id_fkey" FOREIGN KEY ("plan_revision_id") REFERENCES "public"."plan_revisions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "plan_revision_calendar_events_external_id_not_blank" CHECK (btrim(external_event_id) <> ''::text),
  CONSTRAINT "plan_revision_calendar_events_position_non_negative" CHECK ("position" >= 0),
  CONSTRAINT "plan_revision_calendar_events_time_representation" CHECK ((all_day AND (start_date IS NOT NULL) AND (end_date IS NOT NULL) AND (starts_at IS NULL) AND (ends_at IS NULL)) OR ((NOT all_day) AND (starts_at IS NOT NULL) AND (ends_at IS NOT NULL) AND (start_date IS NULL) AND (end_date IS NULL))),
  CONSTRAINT "plan_revision_calendar_events_title_not_blank" CHECK (btrim(title) <> ''::text)
);
-- Create index "plan_revision_calendar_events_calendar_source_id_idx" to table: "plan_revision_calendar_events"
CREATE INDEX "plan_revision_calendar_events_calendar_source_id_idx" ON "public"."plan_revision_calendar_events" ("calendar_source_id");
-- Create index "plan_revision_calendar_events_plan_revision_position_key" to table: "plan_revision_calendar_events"
CREATE UNIQUE INDEX "plan_revision_calendar_events_plan_revision_position_key" ON "public"."plan_revision_calendar_events" ("plan_revision_id", "position");
-- Create "plan_revision_github_issues" table
CREATE TABLE "public"."plan_revision_github_issues" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "plan_revision_id" uuid NOT NULL,
  "repository_owner" text NOT NULL,
  "repository_name" text NOT NULL,
  "issue_number" integer NOT NULL,
  "title" text NOT NULL,
  "html_url" text NOT NULL,
  "position" integer NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_revision_github_issues_plan_revision_id_fkey" FOREIGN KEY ("plan_revision_id") REFERENCES "public"."plan_revisions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "plan_revision_github_issues_name_not_blank" CHECK (btrim(repository_name) <> ''::text),
  CONSTRAINT "plan_revision_github_issues_number_positive" CHECK (issue_number > 0),
  CONSTRAINT "plan_revision_github_issues_owner_not_blank" CHECK (btrim(repository_owner) <> ''::text),
  CONSTRAINT "plan_revision_github_issues_position_non_negative" CHECK ("position" >= 0),
  CONSTRAINT "plan_revision_github_issues_title_not_blank" CHECK (btrim(title) <> ''::text),
  CONSTRAINT "plan_revision_github_issues_url_not_blank" CHECK (btrim(html_url) <> ''::text)
);
-- Create index "plan_revision_github_issues_plan_revision_position_key" to table: "plan_revision_github_issues"
CREATE UNIQUE INDEX "plan_revision_github_issues_plan_revision_position_key" ON "public"."plan_revision_github_issues" ("plan_revision_id", "position");
-- Create "plan_revision_routines" table
CREATE TABLE "public"."plan_revision_routines" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "plan_revision_id" uuid NOT NULL,
  "routine_id" uuid NOT NULL,
  "position" integer NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "plan_revision_routines_plan_revision_id_fkey" FOREIGN KEY ("plan_revision_id") REFERENCES "public"."plan_revisions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "plan_revision_routines_routine_id_fkey" FOREIGN KEY ("routine_id") REFERENCES "public"."routines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "plan_revision_routines_position_non_negative" CHECK ("position" >= 0)
);
-- Create index "plan_revision_routines_plan_revision_position_key" to table: "plan_revision_routines"
CREATE UNIQUE INDEX "plan_revision_routines_plan_revision_position_key" ON "public"."plan_revision_routines" ("plan_revision_id", "position");
-- Create index "plan_revision_routines_routine_id_idx" to table: "plan_revision_routines"
CREATE INDEX "plan_revision_routines_routine_id_idx" ON "public"."plan_revision_routines" ("routine_id");
