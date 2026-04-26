import os
import tempfile

from alt_body.parser import parse_inbody_csv


CSV_HEADER = (
    "日付,測定機器,体重(kg),骨格筋量(kg),筋肉量(kg),体脂肪量(kg),"
    "BMI(kg/m²),体脂肪率(%),基礎代謝量(kcal),InBody点数,"
    "右腕筋肉量(kg),左腕筋肉量(kg),体幹筋肉量(kg),右脚筋肉量(kg),左脚筋肉量(kg),"
    "右腕体脂肪量(kg),左腕体脂肪量(kg),体幹体脂肪量(kg),右脚体脂肪量(kg),左脚体脂肪量(kg),"
    "右腕 細胞外水分比,左腕 細胞外水分比,体幹 細胞外水分比,右脚 細胞外水分比,左脚 細胞外水分比,"
    "ウエストヒップ比,腹囲(cm),内臓脂肪断面積(cm²),内臓脂肪レベル(Level),"
    "体水分量(L),細胞内水分量(L),細胞外水分量(L),細胞外水分比,"
    "上下均衡,上半身均衡,下半身均衡,下肢筋力レベル(Level),下半身筋肉量(kg),"
    "タンパク質量(kg),ミネラル量(kg),骨ミネラル量(kg),体細胞量(kg),"
    "骨格筋指数(kg/m²),全身位相角(°)"
)


def _write_csv(lines: list[str]) -> str:
    f = tempfile.NamedTemporaryFile(
        mode="w", suffix=".csv", delete=False, encoding="utf-8"
    )
    f.write(CSV_HEADER + "\n")
    for line in lines:
        f.write(line + "\n")
    f.close()
    return f.name


def test_parse_single_row():
    path = _write_csv([
        "20260403034553,H30,64.9,29.1,48.6,13.6,21.7,21.0,1478,72.0"
        ",-,-,-,-,-,-,-,-,-,-,-,-,-,-,-,0.82,-,-,4.0,-,-,-,-,1,-,-,-,-,-,-,-,-,-,-"
    ])
    try:
        rows = parse_inbody_csv(path)
        assert len(rows) == 1
        row = rows[0]
        assert row["measured_at"].year == 2026
        assert row["measured_at"].month == 4
        assert row["measured_at"].day == 3
        assert row["weight_kg"] == 64.9
        assert row["skeletal_muscle_mass_kg"] == 29.1
        assert row["muscle_mass_kg"] == 48.6
        assert row["body_fat_mass_kg"] == 13.6
        assert row["bmi"] == 21.7
        assert row["body_fat_percent"] == 21.0
        assert row["basal_metabolic_rate"] == 1478
        assert row["inbody_score"] == 72.0
        assert row["waist_hip_ratio"] == 0.82
        assert row["visceral_fat_level"] == 4
    finally:
        os.unlink(path)


def test_dash_values_become_none():
    path = _write_csv([
        "20260403034553,H30,64.9,-,48.6,13.6,21.7,21.0,1478,72.0"
        ",-,-,-,-,-,-,-,-,-,-,-,-,-,-,-,0.82,-,-,4.0,-,-,-,-,1,-,-,-,-,-,-,-,-,-,-"
    ])
    try:
        rows = parse_inbody_csv(path)
        assert rows[0]["skeletal_muscle_mass_kg"] is None
    finally:
        os.unlink(path)


def test_parse_multiple_rows():
    path = _write_csv([
        "20260403034553,H30,64.9,29.1,48.6,13.6,21.7,21.0,1478,72.0"
        ",-,-,-,-,-,-,-,-,-,-,-,-,-,-,-,0.82,-,-,4.0,-,-,-,-,1,-,-,-,-,-,-,-,-,-,-",
        "20260330063619,H30,66.9,29.6,49.7,14.3,22.4,21.4,1506,72.0"
        ",-,-,-,-,-,-,-,-,-,-,-,-,-,-,-,0.82,-,-,5.0,-,-,-,-,1,-,-,-,-,-,-,-,-,-,-",
    ])
    try:
        rows = parse_inbody_csv(path)
        assert len(rows) == 2
        assert rows[0]["weight_kg"] == 64.9
        assert rows[1]["weight_kg"] == 66.9
    finally:
        os.unlink(path)


def test_empty_csv():
    path = _write_csv([])
    try:
        rows = parse_inbody_csv(path)
        assert rows == []
    finally:
        os.unlink(path)
