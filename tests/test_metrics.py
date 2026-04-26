from alt_body.metrics import calculate_metrics


def test_ffmi_calculation():
    result = calculate_metrics(
        weight_kg=64.9,
        body_fat_percent=21.0,
        skeletal_muscle_mass_kg=29.1,
        height_m=1.70,
    )
    # LBM = 64.9 * (1 - 21.0/100) = 51.271
    # FFMI raw = 51.271 / (1.70^2) = 17.7408
    # FFMI normalized = 17.7408 + 6.1 * (1.8 - 1.70) = 17.7408 + 0.61 = 18.3508 -> 18.35
    assert result["ffmi"] == 18.35
    # SMR = 29.1 / 64.9 * 100 = 44.84
    assert result["skeletal_muscle_ratio"] == 44.84


def test_ffmi_without_body_fat():
    result = calculate_metrics(
        weight_kg=64.9,
        body_fat_percent=None,
        skeletal_muscle_mass_kg=29.1,
        height_m=1.70,
    )
    assert result["ffmi"] is None
    assert result["skeletal_muscle_ratio"] == 44.84


def test_skeletal_muscle_ratio_without_smm():
    result = calculate_metrics(
        weight_kg=64.9,
        body_fat_percent=21.0,
        skeletal_muscle_mass_kg=None,
        height_m=1.70,
    )
    assert result["ffmi"] == 18.35
    assert result["skeletal_muscle_ratio"] is None
