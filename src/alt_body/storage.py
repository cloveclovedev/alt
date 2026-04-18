"""Store body measurements in Neon Postgres."""

from alt_db.connection import NeonHTTP

_COLUMNS = [
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
    """Insert measurements, skipping duplicates. Returns (inserted, skipped)."""
    inserted = 0
    skipped = 0

    for m in measurements:
        params = []
        placeholders = []
        for i, col in enumerate(_COLUMNS):
            value = m[col]
            if hasattr(value, "isoformat"):
                value = value.isoformat()
            params.append(value)
            placeholders.append(f"${i + 1}")

        sql = (
            f"INSERT INTO body_measurements ({', '.join(_COLUMNS)}) "
            f"VALUES ({', '.join(placeholders)}) "
            f"ON CONFLICT (measured_at) DO NOTHING"
        )
        result = db.execute(sql, params)
        if result.row_count > 0:
            inserted += 1
        else:
            skipped += 1

    return inserted, skipped
