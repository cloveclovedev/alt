-- Migrate routine definitions from routines table to entries
-- type='routine_definition', title=name, content=notes, status='active'
-- metadata contains: category, interval_days, active_months, available_days

INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'routine_definition',
    name,
    notes,
    'active',
    jsonb_build_object(
        'category', category,
        'interval_days', interval_days
    )
    || CASE WHEN active_months IS NOT NULL THEN jsonb_build_object('active_months', to_jsonb(active_months)) ELSE '{}'::jsonb END
    || CASE WHEN available_days IS NOT NULL THEN jsonb_build_object('available_days', to_jsonb(available_days)) ELSE '{}'::jsonb END,
    created_at,
    created_at
FROM routines;

-- Drop the routines table
DROP TABLE routines;
