"""CLI entry point for alt-body."""

import argparse
import sys

try:
    import tomllib
except ImportError:
    import tomli as tomllib

from pathlib import Path

from alt_db.connection import NeonHTTP
from .parser import parse_inbody_csv
from .metrics import calculate_metrics
from .storage import upsert_measurements


def _find_config() -> dict:
    """Find and parse alt.toml from the project root."""
    path = Path(__file__).resolve()
    for parent in path.parents:
        toml_path = parent / "alt.toml"
        if toml_path.exists():
            with open(toml_path, "rb") as f:
                return tomllib.load(f)
    return {}


def _run_import(
    db: NeonHTTP, csv_path: str, height_m: float
) -> tuple[int, int, dict | None]:
    """Parse CSV, calculate metrics, store. Returns (inserted, skipped, latest)."""
    rows = parse_inbody_csv(csv_path)
    if not rows:
        return 0, 0, None

    for row in rows:
        metrics = calculate_metrics(
            weight_kg=row["weight_kg"],
            body_fat_percent=row.get("body_fat_percent"),
            skeletal_muscle_mass_kg=row.get("skeletal_muscle_mass_kg"),
            height_m=height_m,
        )
        row["ffmi"] = metrics["ffmi"]
        row["skeletal_muscle_ratio"] = metrics["skeletal_muscle_ratio"]

    inserted, skipped = upsert_measurements(db, rows)

    # Latest by measured_at
    latest = max(rows, key=lambda r: r["measured_at"])
    return inserted, skipped, latest


def main():
    parser = argparse.ArgumentParser(description="Body composition tracking")
    sub = parser.add_subparsers(dest="command")

    import_cmd = sub.add_parser("import", help="Import InBody CSV")
    import_cmd.add_argument("csv_path", help="Path to InBody CSV file")

    args = parser.parse_args()

    if args.command != "import":
        parser.print_help()
        sys.exit(1)

    try:
        config = _find_config()
        height_m = config.get("body", {}).get("height_m")
        if height_m is None:
            print("Error: body.height_m not set in alt.toml", file=sys.stderr)
            sys.exit(1)

        db = NeonHTTP.from_env()
        inserted, skipped, latest = _run_import(db, args.csv_path, height_m)

        print(f"Imported {inserted} new measurements (skipped {skipped} duplicates)")
        if latest:
            ts = latest["measured_at"].strftime("%Y-%m-%d %H:%M")
            wt = latest["weight_kg"]
            bf = latest.get("body_fat_percent", "?")
            ffmi = latest.get("ffmi", "?")
            print(f"Latest: {ts} — {wt}kg / BF {bf}% / FFMI {ffmi}")
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
