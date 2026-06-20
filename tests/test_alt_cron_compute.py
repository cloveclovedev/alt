"""Unit tests for alt_cron.compute_target. Pure function, no DB needed."""

import pytest

from alt_cron import compute_target


def test_empty_rows_raises():
    with pytest.raises(ValueError, match="No time-source params"):
        compute_target([], cron_minute=23)


def test_rows_without_time_sources_raises():
    rows = [
        {"key": "plan.discord.channel_id", "value": "12345", "metadata": {}},
        {"key": "daily_plan.cloud.enabled", "value": True, "metadata": {}},
    ]
    with pytest.raises(ValueError, match="No time-source params"):
        compute_target(rows, cron_minute=23)


def test_single_fallback_hour():
    rows = [
        {"key": "daily_plan.cloud.fallback_hour", "value": 10, "metadata": {}},
    ]
    result = compute_target(rows, cron_minute=23)
    assert result == {
        "minute": 23,
        "hours": [10],
        "cron": "23 10 * * *",
        "warnings": [],
    }


def test_two_fallback_hours():
    rows = [
        {"key": "daily_plan.cloud.fallback_hour", "value": 10, "metadata": {}},
        {"key": "weekly_plan.cloud.fallback_hour", "value": 18, "metadata": {}},
    ]
    result = compute_target(rows, cron_minute=23)
    assert result["hours"] == [10, 18]
    assert result["cron"] == "23 10,18 * * *"


def test_run_hours_array():
    rows = [
        {"key": "x_post.cloud.run_hours", "value": [0, 6, 18], "metadata": {}},
    ]
    result = compute_target(rows, cron_minute=23)
    assert result["hours"] == [0, 6, 18]
    assert result["cron"] == "23 0,6,18 * * *"


def test_mixed_fallback_and_run_dedup():
    rows = [
        {"key": "daily_plan.cloud.fallback_hour", "value": 10, "metadata": {}},
        {"key": "x_post.cloud.run_hours", "value": [6, 10, 18], "metadata": {}},
        {"key": "x_draft.cloud.run_hours", "value": [10, 18], "metadata": {}},
    ]
    result = compute_target(rows, cron_minute=23)
    assert result["hours"] == [6, 10, 18]
    assert result["cron"] == "23 6,10,18 * * *"


def test_fallback_hour_out_of_range():
    rows = [{"key": "x.cloud.fallback_hour", "value": 24, "metadata": {}}]
    with pytest.raises(ValueError, match="x.cloud.fallback_hour.*out of range"):
        compute_target(rows, cron_minute=23)


def test_fallback_hour_negative():
    rows = [{"key": "x.cloud.fallback_hour", "value": -1, "metadata": {}}]
    with pytest.raises(ValueError, match="x.cloud.fallback_hour.*out of range"):
        compute_target(rows, cron_minute=23)


def test_fallback_hour_wrong_type():
    rows = [{"key": "x.cloud.fallback_hour", "value": "10", "metadata": {}}]
    with pytest.raises(ValueError, match="x.cloud.fallback_hour.*expected int"):
        compute_target(rows, cron_minute=23)


def test_fallback_hour_bool_rejected():
    rows = [{"key": "x.cloud.fallback_hour", "value": True, "metadata": {}}]
    with pytest.raises(ValueError, match="x.cloud.fallback_hour.*expected int"):
        compute_target(rows, cron_minute=23)


def test_run_hours_wrong_type():
    rows = [{"key": "x.cloud.run_hours", "value": "6,18", "metadata": {}}]
    with pytest.raises(ValueError, match="x.cloud.run_hours.*expected list"):
        compute_target(rows, cron_minute=23)


def test_run_hours_element_invalid():
    rows = [{"key": "x.cloud.run_hours", "value": [6, 99], "metadata": {}}]
    with pytest.raises(ValueError, match="x.cloud.run_hours.*out of range"):
        compute_target(rows, cron_minute=23)


def test_cron_minute_out_of_range():
    rows = [{"key": "x.cloud.fallback_hour", "value": 10, "metadata": {}}]
    with pytest.raises(ValueError, match="cron_minute out of range"):
        compute_target(rows, cron_minute=60)


def test_cron_minute_wrong_type():
    rows = [{"key": "x.cloud.fallback_hour", "value": 10, "metadata": {}}]
    with pytest.raises(ValueError, match="cron_minute must be int"):
        compute_target(rows, cron_minute="23")  # type: ignore[arg-type]
