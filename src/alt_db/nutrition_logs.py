"""CRUD operations for nutrition_logs table."""


def _row_to_dict(row: tuple) -> dict:
    return {
        "id": str(row[0]),
        "logged_date": str(row[1]),
        "meal_type": row[2],
        "description": row[3],
        "calories_kcal": float(row[4]) if row[4] is not None else None,
        "protein_g": float(row[5]) if row[5] is not None else None,
        "supplement_taken": bool(row[6]),
        "source_message_id": row[7],
        "estimated_by": row[8],
        "created_at": str(row[9]),
        "updated_at": str(row[10]),
    }


def add_log(
    db,
    *,
    logged_date: str,
    meal_type: str,
    description: str,
    calories: float | None = None,
    protein: float | None = None,
    supplement_taken: bool = False,
    source_message_id: str | None = None,
    estimated_by: str = "llm",
) -> dict:
    result = db.execute(
        """INSERT INTO nutrition_logs
        (logged_date, meal_type, description, calories_kcal, protein_g,
         supplement_taken, source_message_id, estimated_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, logged_date, meal_type, description, calories_kcal,
                  protein_g, supplement_taken, source_message_id, estimated_by,
                  created_at, updated_at""",
        [logged_date, meal_type, description, calories, protein,
         supplement_taken, source_message_id, estimated_by],
    )
    return _row_to_dict(result.rows[0])


def list_logs_by_date(db, logged_date: str) -> list[dict]:
    result = db.execute(
        """SELECT id, logged_date, meal_type, description, calories_kcal,
                  protein_g, supplement_taken, source_message_id, estimated_by,
                  created_at, updated_at
        FROM nutrition_logs WHERE logged_date = $1
        ORDER BY created_at""",
        [logged_date],
    )
    return [_row_to_dict(row) for row in result.rows]


def is_message_processed(db, source_message_id: str) -> bool:
    result = db.execute(
        "SELECT COUNT(*) FROM nutrition_logs WHERE source_message_id = $1",
        [source_message_id],
    )
    return result.rows[0][0] > 0


def daily_summary(db, logged_date: str) -> dict:
    result = db.execute(
        """SELECT
            COALESCE(SUM(calories_kcal), 0),
            COALESCE(SUM(protein_g), 0),
            COUNT(*) FILTER (WHERE supplement_taken = true)
        FROM nutrition_logs WHERE logged_date = $1""",
        [logged_date],
    )
    row = result.rows[0]
    return {
        "total_calories": float(row[0]),
        "total_protein": float(row[1]),
        "supplement_taken": int(row[2]) > 0,
    }
