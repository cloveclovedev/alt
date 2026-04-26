"""Store body measurements in Neon Postgres via the entries table."""

import json

from alt_db.connection import NeonHTTP

_MEASUREMENT_FIELDS = [
    "measured_at",
    "weight_kg",
    "skeletal_muscle_mass_kg",
    "muscle_mass_kg",
    "body_fat_mass_kg",
    "body_fat_percent",
    "bmi",
    "basal_metabolic_rate",
    "inbody_score",
    "waist_hip_ratio",
    "visceral_fat_level",
    "ffmi",
    "skeletal_muscle_ratio",
]


def upsert_measurements(
    db: NeonHTTP, measurements: list[dict]
) -> tuple[int, int]:
    """Insert measurements as entries, skipping duplicates. Returns (inserted, skipped)."""
    inserted = 0
    skipped = 0

    for m in measurements:
        measured_at = m["measured_at"]
        if hasattr(measured_at, "isoformat"):
            measured_at = measured_at.isoformat()

        # Check for existing entry with same measured_at
        existing = db.execute(
            "SELECT id FROM entries WHERE type = 'body_measurement' AND metadata->>'measured_at' = $1",
            [measured_at],
        )
        if existing.rows:
            skipped += 1
            continue

        # Build metadata from measurement fields
        metadata = {}
        for field in _MEASUREMENT_FIELDS:
            value = m[field]
            if hasattr(value, "isoformat"):
                value = value.isoformat()
            metadata[field] = value

        title = "InBody " + str(measured_at)[:10]

        db.execute(
            "INSERT INTO entries (type, title, metadata) VALUES ($1, $2, $3)",
            ["body_measurement", title, json.dumps(metadata)],
        )
        inserted += 1

    return inserted, skipped
