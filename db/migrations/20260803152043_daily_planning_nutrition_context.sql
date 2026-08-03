-- Modify "daily_planning_contexts" table
ALTER TABLE "public"."daily_planning_contexts" ADD COLUMN "nutrition_context" jsonb NOT NULL DEFAULT '{}', ADD COLUMN "nutrition_status" text NOT NULL DEFAULT 'unavailable';
