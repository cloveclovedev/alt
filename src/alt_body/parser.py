"""Parse InBody CSV exports into structured measurement dicts."""

import csv
from datetime import datetime, timezone, timedelta

JST = timezone(timedelta(hours=9))

# Map CSV column headers to (db_column, type)
_COLUMN_MAP = {
    "体重(kg)": ("weight_kg", float),
    "骨格筋量(kg)": ("skeletal_muscle_mass_kg", float),
    "筋肉量(kg)": ("muscle_mass_kg", float),
    "体脂肪量(kg)": ("body_fat_mass_kg", float),
    "体脂肪率(%)": ("body_fat_percent", float),
    "BMI(kg/m²)": ("bmi", float),
    "基礎代謝量(kcal)": ("basal_metabolic_rate", int),
    "InBody点数": ("inbody_score", float),
    "ウエストヒップ比": ("waist_hip_ratio", float),
    "内臓脂肪レベル(Level)": ("visceral_fat_level", int),
}


def _parse_value(raw: str, type_fn: type) -> float | int | None:
    """Parse a CSV cell value, returning None for '-' or empty."""
    stripped = raw.strip()
    if stripped == "-" or stripped == "":
        return None
    if type_fn is int:
        return int(float(stripped))
    return type_fn(stripped)


def _parse_timestamp(raw: str) -> datetime:
    """Parse InBody timestamp format YYYYMMDDHHmmss to datetime with JST."""
    return datetime.strptime(raw.strip(), "%Y%m%d%H%M%S").replace(tzinfo=JST)


def parse_inbody_csv(path: str) -> list[dict]:
    """Parse an InBody CSV file and return a list of measurement dicts."""
    rows = []
    with open(path, encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        for csv_row in reader:
            measurement = {
                "measured_at": _parse_timestamp(csv_row["日付"]),
            }
            for csv_col, (db_col, type_fn) in _COLUMN_MAP.items():
                measurement[db_col] = _parse_value(csv_row[csv_col], type_fn)
            rows.append(measurement)
    return rows
