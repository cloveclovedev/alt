"""Tests for body measurement storage."""

from datetime import datetime, timezone, timedelta

from alt_body.storage import upsert_measurements

JST = timezone(timedelta(hours=9))


def test_upsert_single_measurement(db):
    client, created_ids, *_ = db
    ts = datetime(2099, 1, 1, 12, 0, 0, tzinfo=JST)
    ts_iso = ts.isoformat()

    measurements = [
        {
            "measured_at": ts,
            "weight_kg": 64.9,
            "skeletal_muscle_mass_kg": 29.1,
            "muscle_mass_kg": 48.6,
            "body_fat_mass_kg": 13.6,
            "body_fat_percent": 21.0,
            "bmi": 21.7,
            "basal_metabolic_rate": 1478,
            "inbody_score": 72.0,
            "waist_hip_ratio": 0.82,
            "visceral_fat_level": 4,
            "ffmi": 17.56,
            "skeletal_muscle_ratio": 44.84,
        }
    ]
    inserted, skipped = upsert_measurements(client, measurements)
    assert inserted == 1
    assert skipped == 0

    # Track inserted entry IDs for cleanup
    result = client.execute(
        "SELECT id FROM entries WHERE type = 'body_measurement' AND metadata->>'measured_at' = $1",
        [ts_iso],
    )
    for row in result.rows:
        created_ids.append(row[0])

    # Verify data in DB via entries table
    result = client.execute(
        "SELECT metadata FROM entries WHERE type = 'body_measurement' AND metadata->>'measured_at' = $1",
        [ts_iso],
    )
    assert len(result.rows) == 1
    metadata = result.rows[0][0]
    if isinstance(metadata, str):
        import json
        metadata = json.loads(metadata)
    assert float(metadata["weight_kg"]) == 64.9
    assert float(metadata["ffmi"]) == 17.56


def test_upsert_skips_duplicates(db):
    client, created_ids, *_ = db
    ts = datetime(2099, 1, 2, 12, 0, 0, tzinfo=JST)
    ts_iso = ts.isoformat()

    measurements = [
        {
            "measured_at": ts,
            "weight_kg": 64.9,
            "skeletal_muscle_mass_kg": None,
            "muscle_mass_kg": None,
            "body_fat_mass_kg": None,
            "body_fat_percent": None,
            "bmi": None,
            "basal_metabolic_rate": None,
            "inbody_score": None,
            "waist_hip_ratio": None,
            "visceral_fat_level": None,
            "ffmi": None,
            "skeletal_muscle_ratio": None,
        }
    ]
    inserted1, skipped1 = upsert_measurements(client, measurements)
    inserted2, skipped2 = upsert_measurements(client, measurements)

    assert inserted1 == 1
    assert skipped1 == 0
    assert inserted2 == 0
    assert skipped2 == 1

    # Track inserted entry IDs for cleanup
    result = client.execute(
        "SELECT id FROM entries WHERE type = 'body_measurement' AND metadata->>'measured_at' = $1",
        [ts_iso],
    )
    for row in result.rows:
        created_ids.append(row[0])

    # Verify only one entry exists
    assert len(result.rows) == 1
