"""CRUD operations for nutrition_items table."""


def _row_to_dict(row: tuple) -> dict:
    return {
        "id": str(row[0]),
        "name": row[1],
        "calories_kcal": float(row[2]),
        "protein_g": float(row[3]),
        "source": row[4],
        "created_at": str(row[5]),
        "updated_at": str(row[6]),
    }


def add_item(db, *, name: str, calories: float, protein: float, source: str = "user_registered") -> dict:
    result = db.execute(
        """INSERT INTO nutrition_items (name, calories_kcal, protein_g, source)
        VALUES ($1, $2, $3, $4)
        RETURNING id, name, calories_kcal, protein_g, source, created_at, updated_at""",
        [name, calories, protein, source],
    )
    return _row_to_dict(result.rows[0])


def get_item_by_name(db, name: str) -> dict | None:
    result = db.execute(
        """SELECT id, name, calories_kcal, protein_g, source, created_at, updated_at
        FROM nutrition_items WHERE name = $1""",
        [name],
    )
    return _row_to_dict(result.rows[0]) if result.rows else None


def list_items(db) -> list[dict]:
    result = db.execute(
        """SELECT id, name, calories_kcal, protein_g, source, created_at, updated_at
        FROM nutrition_items ORDER BY name""",
    )
    return [_row_to_dict(row) for row in result.rows]


def update_item(db, name: str, **kwargs) -> bool:
    sets = []
    params = []
    idx = 1
    for key in ("calories_kcal", "protein_g", "source"):
        field = key.replace("calories_kcal", "calories").replace("protein_g", "protein")
        if field in kwargs and kwargs[field] is not None:
            sets.append(f"{key} = ${idx}")
            params.append(kwargs[field])
            idx += 1
    if not sets:
        return False
    sets.append(f"updated_at = now()")
    params.append(name)
    result = db.execute(
        f"UPDATE nutrition_items SET {', '.join(sets)} WHERE name = ${idx}",
        params,
    )
    return result.row_count > 0


def delete_item(db, name: str) -> bool:
    result = db.execute("DELETE FROM nutrition_items WHERE name = $1", [name])
    return result.row_count > 0
