"""CRUD operations for nutrition_targets table."""


def _row_to_dict(row: tuple) -> dict:
    return {
        "id": str(row[0]),
        "calories_kcal": float(row[1]),
        "protein_g": float(row[2]),
        "effective_from": str(row[3]),
        "effective_until": str(row[4]) if row[4] is not None else None,
        "rationale": row[5],
        "created_at": str(row[6]),
        "updated_at": str(row[7]),
    }


def add_target(
    db,
    *,
    calories: float,
    protein: float,
    effective_from: str,
    rationale: str | None = None,
) -> dict:
    # Deactivate any current target
    db.execute(
        """UPDATE nutrition_targets SET effective_until = $1, updated_at = now()
        WHERE effective_until IS NULL""",
        [effective_from],
    )
    result = db.execute(
        """INSERT INTO nutrition_targets (calories_kcal, protein_g, effective_from, rationale)
        VALUES ($1, $2, $3, $4)
        RETURNING id, calories_kcal, protein_g, effective_from, effective_until,
                  rationale, created_at, updated_at""",
        [calories, protein, effective_from, rationale],
    )
    return _row_to_dict(result.rows[0])


def get_active_target(db, date: str) -> dict | None:
    result = db.execute(
        """SELECT id, calories_kcal, protein_g, effective_from, effective_until,
                  rationale, created_at, updated_at
        FROM nutrition_targets
        WHERE effective_from <= $1 AND (effective_until IS NULL OR effective_until > $1)
        ORDER BY effective_from DESC LIMIT 1""",
        [date],
    )
    return _row_to_dict(result.rows[0]) if result.rows else None


def list_targets(db) -> list[dict]:
    result = db.execute(
        """SELECT id, calories_kcal, protein_g, effective_from, effective_until,
                  rationale, created_at, updated_at
        FROM nutrition_targets ORDER BY effective_from DESC""",
    )
    return [_row_to_dict(row) for row in result.rows]


def deactivate_target(db, target_id: str, effective_until: str) -> bool:
    result = db.execute(
        """UPDATE nutrition_targets SET effective_until = $1, updated_at = now()
        WHERE id = $2""",
        [effective_until, target_id],
    )
    return result.row_count > 0
