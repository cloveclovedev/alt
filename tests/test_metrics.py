from alt_body.metrics import calculate_metrics


def test_ffmi_calculation():
    result = calculate_metrics(
        weight_kg=64.9,
        body_fat_percent=21.0,
        skeletal_muscle_mass_kg=29.1,
        height_m=1.73,
    )
    # LBM = 64.9 * (1 - 21.0/100) = 51.271
    # FFMI raw = 51.271 / (1.73^2) = 17.1309
    # FFMI normalized = 17.1309 + 6.1 * (1.8 - 1.73) = 17.1309 + 0.427 = 17.558 -> 17.56
    assert result["ffmi"] == 17.56
    # SMR = 29.1 / 64.9 * 100 = 44.84
    assert result["skeletal_muscle_ratio"] == 44.84


def test_ffmi_without_body_fat():
    result = calculate_metrics(
        weight_kg=64.9,
        body_fat_percent=None,
        skeletal_muscle_mass_kg=29.1,
        height_m=1.73,
    )
    assert result["ffmi"] is None
    assert result["skeletal_muscle_ratio"] == 44.84


def test_skeletal_muscle_ratio_without_smm():
    result = calculate_metrics(
        weight_kg=64.9,
        body_fat_percent=21.0,
        skeletal_muscle_mass_kg=None,
        height_m=1.73,
    )
    assert result["ffmi"] == 17.56
    assert result["skeletal_muscle_ratio"] is None
