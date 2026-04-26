"""Body composition metric calculations."""


def calculate_metrics(
    weight_kg: float,
    body_fat_percent: float | None,
    skeletal_muscle_mass_kg: float | None,
    height_m: float,
) -> dict:
    """Calculate FFMI and skeletal muscle ratio from measurements."""
    ffmi = None
    if body_fat_percent is not None:
        lbm = weight_kg * (1 - body_fat_percent / 100)
        ffmi_raw = lbm / (height_m**2)
        ffmi = round(ffmi_raw + 6.1 * (1.8 - height_m), 2)

    skeletal_muscle_ratio = None
    if skeletal_muscle_mass_kg is not None:
        skeletal_muscle_ratio = round(
            (skeletal_muscle_mass_kg / weight_kg) * 100, 2
        )

    return {
        "ffmi": ffmi,
        "skeletal_muscle_ratio": skeletal_muscle_ratio,
    }
