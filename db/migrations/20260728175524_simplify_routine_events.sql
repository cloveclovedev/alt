-- Modify "routine_events" table
ALTER TABLE "public"."routine_events" DROP COLUMN "kind";
-- Drop enum type "routine_event_kind"
DROP TYPE "public"."routine_event_kind";
