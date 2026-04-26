-- Universal entries consolidation: merge 6 tables into entries
-- See docs/superpowers/specs/2026-04-22-universal-entries-design.md

-- 1. Add parent_id column
ALTER TABLE entries ADD COLUMN parent_id uuid REFERENCES entries(id) ON DELETE SET NULL;
CREATE INDEX idx_entries_parent ON entries (parent_id) WHERE parent_id IS NOT NULL;

-- 2. Preserve existing tags in metadata before dropping
UPDATE entries
SET metadata = jsonb_set(COALESCE(metadata, '{}'), '{tags}', COALESCE(tags, '[]'))
WHERE tags IS NOT NULL AND tags != '[]'::jsonb;

-- 3. Drop tags column and its GIN index
DROP INDEX IF EXISTS idx_entries_tags;
ALTER TABLE entries DROP COLUMN tags;

-- 4. Migrate routine_events → entries (type='routine_event')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    re.id,
    'routine_event',
    r.name,
    NULL,
    NULL,
    jsonb_build_object(
        'category', r.category,
        'completed_at', re.completed_at,
        'routine_id', re.routine_id
    ),
    re.completed_at,
    re.completed_at
FROM routine_events re
JOIN routines r ON r.id = re.routine_id;

-- 5. Migrate body_measurements → entries (type='body_measurement')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'body_measurement',
    'InBody ' || to_char(measured_at AT TIME ZONE 'Asia/Tokyo', 'YYYY-MM-DD'),
    NULL,
    NULL,
    jsonb_build_object(
        'measured_at', measured_at,
        'weight_kg', weight_kg,
        'skeletal_muscle_mass_kg', skeletal_muscle_mass_kg,
        'muscle_mass_kg', muscle_mass_kg,
        'body_fat_mass_kg', body_fat_mass_kg,
        'body_fat_percent', body_fat_percent,
        'bmi', bmi,
        'basal_metabolic_rate', basal_metabolic_rate,
        'inbody_score', inbody_score,
        'waist_hip_ratio', waist_hip_ratio,
        'visceral_fat_level', visceral_fat_level,
        'ffmi', ffmi,
        'skeletal_muscle_ratio', skeletal_muscle_ratio,
        'source', source
    ),
    created_at,
    created_at
FROM body_measurements;

-- 6. Migrate body_measurement_goals → entries (type='body_measurement_goal')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'body_measurement_goal',
    metric,
    NULL,
    status,
    jsonb_build_object(
        'metric', metric,
        'target_value', target_value,
        'start_value', start_value,
        'start_date', start_date,
        'target_date', target_date
    ),
    created_at,
    created_at
FROM body_measurement_goals;

-- 7. Migrate nutrition_items → entries (type='nutrition_item')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'nutrition_item',
    name,
    NULL,
    NULL,
    jsonb_build_object(
        'calories_kcal', calories_kcal,
        'protein_g', protein_g,
        'source', source
    ),
    created_at,
    updated_at
FROM nutrition_items;

-- 8. Migrate nutrition_logs → entries (type='nutrition_log')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'nutrition_log',
    description,
    NULL,
    NULL,
    jsonb_build_object(
        'logged_date', logged_date,
        'meal_type', meal_type,
        'calories_kcal', calories_kcal,
        'protein_g', protein_g,
        'supplement_taken', supplement_taken,
        'source_message_id', source_message_id,
        'estimated_by', estimated_by
    ),
    created_at,
    updated_at
FROM nutrition_logs;

-- 9. Migrate nutrition_targets → entries (type='nutrition_target')
INSERT INTO entries (id, type, title, content, status, metadata, created_at, updated_at)
SELECT
    id,
    'nutrition_target',
    'Nutrition Target ' || effective_from,
    rationale,
    CASE WHEN effective_until IS NULL THEN 'active' ELSE 'inactive' END,
    jsonb_build_object(
        'calories_kcal', calories_kcal,
        'protein_g', protein_g,
        'effective_from', effective_from,
        'effective_until', effective_until
    ),
    created_at,
    updated_at
FROM nutrition_targets;

-- 10. Drop old tables
DROP TABLE routine_events;
DROP TABLE body_measurement_goals;
DROP TABLE body_measurements;
DROP TABLE nutrition_logs;
DROP TABLE nutrition_items;
DROP TABLE nutrition_targets;
