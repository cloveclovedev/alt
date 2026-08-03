-- Make "ai_generations" feature-neutral: usage metadata is now scoped to a user
-- (internal/ai owns it) instead of a daily planning session. Backfill user_id
-- from the referencing session before enforcing NOT NULL; the old session FK was
-- ON DELETE CASCADE, so no orphaned rows exist.
ALTER TABLE "public"."ai_generations" ADD COLUMN "user_id" uuid;
UPDATE "public"."ai_generations" AS g
  SET "user_id" = s."user_id"
  FROM "public"."daily_planning_sessions" AS s
  WHERE s."id" = g."session_id";
ALTER TABLE "public"."ai_generations" ALTER COLUMN "user_id" SET NOT NULL;
ALTER TABLE "public"."ai_generations"
  ADD CONSTRAINT "ai_generations_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
ALTER TABLE "public"."ai_generations" DROP COLUMN "session_id";
-- Create index "ai_generations_user_id_created_at_idx" to table: "ai_generations"
CREATE INDEX "ai_generations_user_id_created_at_idx" ON "public"."ai_generations" ("user_id", "created_at");
