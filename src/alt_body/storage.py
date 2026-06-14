"""Store body measurements in Neon Postgres via the entries table."""

import json
from datetime import datetime, timezone

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


def _to_utc_iso(value: datetime) -> str:
    """Serialize an aware datetime to canonical UTC ISO-8601 (offset +00:00)."""
    return value.astimezone(timezone.utc).isoformat()


def upsert_measurements(
    db: NeonHTTP, measurements: list[dict]
) -> tuple[int, int]:
    """Insert measurements as entries, skipping duplicates. Returns (inserted, skipped).

    Duplicate detection compares ``metadata.measured_at`` as ``timestamptz`` so legacy
    rows stored with a different offset (e.g. JST vs UTC) still dedup correctly.
    All newly written rows store ``measured_at`` in canonical UTC ISO form.
    """
    inserted = 0
    skipped = 0

    for m in measurements:
        measured_at_dt = m["measured_at"]
        measured_at_utc = _to_utc_iso(measured_at_dt)

        existing = db.execute(
            "SELECT id FROM entries WHERE type = 'body_measurement' "
            "AND (metadata->>'measured_at')::timestamptz = $1::timestamptz",
            [measured_at_utc],
        )
        if existing.rows:
            skipped += 1
            continue

        metadata = {}
        for field in _MEASUREMENT_FIELDS:
            value = m[field]
            if isinstance(value, datetime):
                value = _to_utc_iso(value)
            metadata[field] = value

        # Title preserves the input-local date (parser emits JST), regardless of UTC storage.
        title = "InBody " + measured_at_dt.strftime("%Y-%m-%d")

        db.execute(
            "INSERT INTO entries (type, title, metadata) VALUES ($1, $2, $3)",
            ["body_measurement", title, json.dumps(metadata)],
        )
        inserted += 1

    return inserted, skipped
