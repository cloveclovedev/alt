-- Create "entries" table
CREATE TABLE "entries" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "type" text NOT NULL,
  "title" text NOT NULL,
  "content" text NULL,
  "status" text NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "parent_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_entries_parent" FOREIGN KEY ("parent_id") REFERENCES "entries" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Create index "idx_entries_created" to table: "entries"
CREATE INDEX "idx_entries_created" ON "entries" ("created_at");
-- Create index "idx_entries_parent" to table: "entries"
CREATE INDEX "idx_entries_parent" ON "entries" ("parent_id") WHERE (parent_id IS NOT NULL);
-- Create index "idx_entries_type" to table: "entries"
CREATE INDEX "idx_entries_type" ON "entries" ("type");
