import os
import tempfile

from alt_body.cli import _run_import


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

CSV_ROW = (
    "20990101120000,H30,64.9,29.1,48.6,13.6,21.7,21.0,1478,72.0"
    ",-,-,-,-,-,-,-,-,-,-,-,-,-,-,-,0.82,-,-,4.0,-,-,-,-,1,-,-,-,-,-,-,-,-,-,-"
)


def test_run_import_parses_and_enriches(db):
    client, created_ids = db
    path = tempfile.NamedTemporaryFile(
        mode="w", suffix=".csv", delete=False, encoding="utf-8"
    )
    path.write(CSV_HEADER + "\n" + CSV_ROW + "\n")
    path.close()

    try:
        inserted, skipped, latest = _run_import(client, path.name, height_m=1.73)
        assert inserted == 1
        assert skipped == 0
        assert latest is not None
        assert latest["weight_kg"] == 64.9
        assert latest["ffmi"] == 17.56
        assert latest["skeletal_muscle_ratio"] == 44.84
    finally:
        os.unlink(path.name)

    # Track inserted entry for cleanup
    result = client.execute(
        "SELECT id FROM entries WHERE type = 'body_measurement' AND metadata->>'measured_at' = $1",
        ["2099-01-01T12:00:00+09:00"],
    )
    for row in result.rows:
        created_ids.append(row[0])
