"""Subprocess tests for the alt-cron CLI."""

import json
import subprocess


def _run(stdin: str, *args: str) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["uv", "run", "alt-cron", *args],
        input=stdin,
        capture_output=True,
        text=True,
    )


def test_cli_compute_basic():
    rows = [{"key": "daily_plan.cloud.fallback_hour", "value": 10, "metadata": {}}]
    result = _run(json.dumps(rows), "compute", "--cron-minute", "23")
    assert result.returncode == 0, result.stderr
    parsed = json.loads(result.stdout)
    assert parsed == {
        "minute": 23,
        "hours": [10],
        "cron": "23 10 * * *",
        "warnings": [],
    }


def test_cli_compute_invalid_cron_minute_exits_nonzero():
    rows = [{"key": "x.cloud.fallback_hour", "value": 10, "metadata": {}}]
    result = _run(json.dumps(rows), "compute", "--cron-minute", "60")
    assert result.returncode != 0
    assert "cron_minute" in result.stderr


def test_cli_compute_empty_rows_exits_nonzero():
    result = _run("[]", "compute", "--cron-minute", "23")
    assert result.returncode != 0
    assert "No time-source params" in result.stderr


def test_cli_compute_malformed_json_exits_nonzero():
    result = _run("not-json", "compute", "--cron-minute", "23")
    assert result.returncode != 0
