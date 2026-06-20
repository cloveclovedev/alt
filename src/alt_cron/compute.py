"""Pure function that derives the unified cron from config rows."""


_FALLBACK_HOUR_SUFFIX = ".cloud.fallback_hour"
_RUN_HOURS_SUFFIX = ".cloud.run_hours"


def compute_target(rows: list[dict], cron_minute: int) -> dict:
    """Compute the target cron expression from time-source config rows.

    Args:
      rows: output of `alt-db config list --with-meta --json` (list of
            objects with `key`, `value`, `metadata` fields).
      cron_minute: integer 0-59, the minute slot used by the unified trigger.

    Returns:
      {"minute": int, "hours": list[int], "cron": str, "warnings": list[str]}

    Raises:
      ValueError: on invalid input (bad cron_minute, malformed time-source
                  values, or empty hour set).
    """
    if not isinstance(cron_minute, int) or isinstance(cron_minute, bool):
        raise ValueError(f"cron_minute must be int, got {type(cron_minute).__name__}")
    if not 0 <= cron_minute <= 59:
        raise ValueError(f"cron_minute out of range (0-59): {cron_minute}")

    hours: set[int] = set()
    for row in rows:
        key = row.get("key", "")
        value = row.get("value")
        if key.endswith(_FALLBACK_HOUR_SUFFIX):
            hours.add(_validate_hour(key, value))
        elif key.endswith(_RUN_HOURS_SUFFIX):
            if not isinstance(value, list):
                raise ValueError(f"{key}: expected list, got {type(value).__name__}")
            for item in value:
                hours.add(_validate_hour(key, item))

    if not hours:
        raise ValueError("No time-source params found in config rows")

    sorted_hours = sorted(hours)
    cron = f"{cron_minute} {','.join(str(h) for h in sorted_hours)} * * *"
    return {
        "minute": cron_minute,
        "hours": sorted_hours,
        "cron": cron,
        "warnings": [],
    }


def _validate_hour(key: str, value: object) -> int:
    if isinstance(value, bool) or not isinstance(value, int):
        raise ValueError(f"{key}: expected int hour, got {type(value).__name__}")
    if not 0 <= value <= 23:
        raise ValueError(f"{key}: hour out of range (0-23): {value}")
    return value
