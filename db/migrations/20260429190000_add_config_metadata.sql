-- db/migrations/20260429190000_add_config_metadata.sql
-- Add a metadata jsonb column to config so each key can carry its own
-- description / type / consumed_by / default. Existing rows default to '{}'.

ALTER TABLE "config"
  ADD COLUMN "metadata" jsonb NOT NULL DEFAULT '{}'::jsonb;
