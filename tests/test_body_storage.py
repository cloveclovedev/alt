"""Tests for body measurement storage."""

from datetime import datetime, timezone, timedelta

from alt_body.storage import upsert_measurements

JST = timezone(timedelta(hours=9))


def test_upsert_single_measurement(db):
    client, _, _, body_timestamps = db
    ts = datetime(2099, 1, 1, 12, 0, 0, tzinfo=JST)
    body_timestamps.append(ts.isoformat())

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

    # Verify data in DB
    result = client.execute(
        "SELECT weight_kg, ffmi FROM body_measurements WHERE measured_at = $1",
        [ts.isoformat()],
    )
    assert len(result.rows) == 1
    assert float(result.rows[0][0]) == 64.9
    assert float(result.rows[0][1]) == 17.56


def test_upsert_skips_duplicates(db):
    client, _, _, body_timestamps = db
    ts = datetime(2099, 1, 2, 12, 0, 0, tzinfo=JST)
    body_timestamps.append(ts.isoformat())

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
    inserted1, _ = upsert_measurements(client, measurements)
    inserted2, skipped2 = upsert_measurements(client, measurements)
    assert inserted1 == 1
    assert inserted2 == 0
    assert skipped2 == 1
