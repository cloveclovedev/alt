import { sql } from "./db"
import type { BodyMeasurement, BodyGoal, BodyGoalStatus } from "./types"

function serializeDate(value: unknown): string {
  if (value instanceof Date) return value.toISOString()
  return String(value ?? "")
}

function serializeDateOnly(value: unknown): string {
  if (value instanceof Date) {
    const y = value.getFullYear()
    const m = String(value.getMonth() + 1).padStart(2, "0")
    const d = String(value.getDate()).padStart(2, "0")
    return `${y}-${m}-${d}`
  }
  return String(value ?? "")
}

function serializeGoal(row: Record<string, unknown>): BodyGoal {
  return {
    ...row,
    start_date: serializeDateOnly(row.start_date),
    target_date: serializeDateOnly(row.target_date),
    created_at: serializeDate(row.created_at),
  } as BodyGoal
}

export async function getBodyMeasurements(
  startDate: string | null
): Promise<BodyMeasurement[]> {
  if (startDate) {
    const rows = await sql`
      SELECT * FROM body_measurements
      WHERE measured_at >= ${startDate}::date
      ORDER BY measured_at ASC
    `
    return rows as BodyMeasurement[]
  }
  const rows = await sql`
    SELECT * FROM body_measurements
    ORDER BY measured_at ASC
  `
  return rows as BodyMeasurement[]
}

export async function getLatestMeasurement(): Promise<BodyMeasurement | null> {
  const rows = await sql`
    SELECT * FROM body_measurements
    ORDER BY measured_at DESC
    LIMIT 1
  `
  return (rows[0] as BodyMeasurement) ?? null
}

export async function getBodyGoals(
  status: BodyGoalStatus | null = null
): Promise<BodyGoal[]> {
  if (status) {
    const rows = await sql`
      SELECT * FROM body_measurement_goals
      WHERE status = ${status}
      ORDER BY created_at DESC
    `
    return rows.map((r) => serializeGoal(r as Record<string, unknown>))
  }
  const rows = await sql`
    SELECT * FROM body_measurement_goals
    ORDER BY created_at DESC
  `
  return rows.map((r) => serializeGoal(r as Record<string, unknown>))
}

export async function createBodyGoal(input: {
  metric: string
  target_value: number
  target_date: string
}): Promise<BodyGoal> {
  const latest = await getLatestMeasurement()
  const startValue = latest
    ? (latest[input.metric as keyof BodyMeasurement] as number | null)
    : null
  const startDate = new Date().toISOString().slice(0, 10)

  const rows = await sql`
    INSERT INTO body_measurement_goals (metric, target_value, start_value, start_date, target_date)
    VALUES (${input.metric}, ${input.target_value}, ${startValue}, ${startDate}, ${input.target_date})
    RETURNING *
  `
  return serializeGoal(rows[0] as Record<string, unknown>)
}

export async function updateBodyGoal(
  id: string,
  input: { target_value?: number; target_date?: string; status?: BodyGoalStatus }
): Promise<BodyGoal> {
  const rows = await sql`
    UPDATE body_measurement_goals
    SET
      target_value = COALESCE(${input.target_value ?? null}::decimal, target_value),
      target_date = COALESCE(${input.target_date ?? null}::date, target_date),
      status = COALESCE(${input.status ?? null}::text, status)
    WHERE id = ${id}
    RETURNING *
  `
  return serializeGoal(rows[0] as Record<string, unknown>)
}

export async function getAllMeasurementsForCalibration(): Promise<
  { weight_kg: number; muscle_mass_kg: number | null; skeletal_muscle_mass_kg: number | null; body_fat_mass_kg: number | null }[]
> {
  const rows = await sql`
    SELECT weight_kg, muscle_mass_kg, skeletal_muscle_mass_kg, body_fat_mass_kg
    FROM body_measurements
    ORDER BY measured_at ASC
  `
  return rows as { weight_kg: number; muscle_mass_kg: number | null; skeletal_muscle_mass_kg: number | null; body_fat_mass_kg: number | null }[]
}

export async function getMeasurementHistory(
  limit: number = 20,
  offset: number = 0
): Promise<{ rows: BodyMeasurement[]; total: number }> {
  const countResult = await sql`
    SELECT COUNT(*)::int as total FROM body_measurements
  `
  const rows = await sql`
    SELECT * FROM body_measurements
    ORDER BY measured_at DESC
    LIMIT ${limit} OFFSET ${offset}
  `
  return {
    rows: rows as BodyMeasurement[],
    total: (countResult[0] as { total: number }).total,
  }
}
