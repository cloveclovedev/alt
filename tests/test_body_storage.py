"""Tests for body measurement storage."""

import json
from datetime import datetime, timezone, timedelta

from alt_body.storage import upsert_measurements

JST = timezone(timedelta(hours=9))
UTC = timezone.utc


_EMPTY_FIELDS = {
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


def test_upsert_single_measurement(db):
    client, created_ids = db
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

    result = client.execute(
        "SELECT id, metadata FROM entries WHERE type = 'body_measurement' "
        "AND (metadata->>'measured_at')::timestamptz = $1::timestamptz",
        [ts_iso],
    )
    assert len(result.rows) == 1
    created_ids.append(result.rows[0][0])
    metadata = result.rows[0][1]
    if isinstance(metadata, str):
        metadata = json.loads(metadata)
    assert float(metadata["weight_kg"]) == 64.9
    assert float(metadata["ffmi"]) == 17.56


def test_upsert_skips_duplicates(db):
    client, created_ids = db
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

    result = client.execute(
        "SELECT id FROM entries WHERE type = 'body_measurement' "
        "AND (metadata->>'measured_at')::timestamptz = $1::timestamptz",
        [ts_iso],
    )
    assert len(result.rows) == 1
    created_ids.append(result.rows[0][0])


def test_upsert_stores_measured_at_normalized_to_utc(db):
    """measured_at stored in UTC ISO; title keeps the input-local (JST) date even when
    that date differs from the UTC date (early-morning JST measurements cross UTC midnight).
    """
    client, created_ids = db
    ts_jst = datetime(2099, 1, 4, 3, 0, 0, tzinfo=JST)  # JST 03:00 -> UTC prev-day 18:00

    measurements = [{"measured_at": ts_jst, "weight_kg": 64.9, **_EMPTY_FIELDS}]
    upsert_measurements(client, measurements)

    result = client.execute(
        "SELECT id, metadata->>'measured_at' FROM entries "
        "WHERE type = 'body_measurement' AND title = $1",
        ["InBody 2099-01-04"],
    )
    assert len(result.rows) == 1
    created_ids.append(result.rows[0][0])
    stored_measured_at = result.rows[0][1]
    assert stored_measured_at == "2099-01-03T18:00:00+00:00"


def test_upsert_skips_duplicates_across_timezone_formats(db):
    """Legacy entries stored with a different tz offset must still dedup with new inputs."""
    client, created_ids = db

    legacy_jst_str = "2099-01-05T12:00:00+09:00"
    legacy_metadata = {"measured_at": legacy_jst_str, "weight_kg": 64.9, **_EMPTY_FIELDS}
    result = client.execute(
        "INSERT INTO entries (type, title, metadata) VALUES ($1, $2, $3) RETURNING id",
        ["body_measurement", "InBody 2099-01-05", json.dumps(legacy_metadata)],
    )
    created_ids.append(result.rows[0][0])

    ts_utc_same_moment = datetime(2099, 1, 5, 3, 0, 0, tzinfo=UTC)
    measurements = [
        {"measured_at": ts_utc_same_moment, "weight_kg": 64.9, **_EMPTY_FIELDS}
    ]
    inserted, skipped = upsert_measurements(client, measurements)

    assert inserted == 0
    assert skipped == 1
