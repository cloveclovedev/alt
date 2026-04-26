import { sql } from "./db"
import type { BodyMeasurement, BodyGoal, BodyGoalStatus } from "./types"

function serializeDateOnly(value: unknown): string {
  if (value instanceof Date) {
    const y = value.getFullYear()
    const m = String(value.getMonth() + 1).padStart(2, "0")
    const d = String(value.getDate()).padStart(2, "0")
    return `${y}-${m}-${d}`
  }
  return String(value ?? "")
}

function entryToMeasurement(row: Record<string, unknown>): BodyMeasurement {
  const meta = (row.metadata ?? {}) as Record<string, unknown>
  return {
    id: row.id as string,
    measured_at: String(meta.measured_at ?? ""),
    weight_kg: Number(meta.weight_kg ?? 0),
    skeletal_muscle_mass_kg: meta.skeletal_muscle_mass_kg != null ? Number(meta.skeletal_muscle_mass_kg) : null,
    muscle_mass_kg: meta.muscle_mass_kg != null ? Number(meta.muscle_mass_kg) : null,
    body_fat_mass_kg: meta.body_fat_mass_kg != null ? Number(meta.body_fat_mass_kg) : null,
    body_fat_percent: meta.body_fat_percent != null ? Number(meta.body_fat_percent) : null,
    bmi: meta.bmi != null ? Number(meta.bmi) : null,
    basal_metabolic_rate: meta.basal_metabolic_rate != null ? Number(meta.basal_metabolic_rate) : null,
    inbody_score: meta.inbody_score != null ? Number(meta.inbody_score) : null,
    waist_hip_ratio: meta.waist_hip_ratio != null ? Number(meta.waist_hip_ratio) : null,
    visceral_fat_level: meta.visceral_fat_level != null ? Number(meta.visceral_fat_level) : null,
    ffmi: meta.ffmi != null ? Number(meta.ffmi) : null,
    skeletal_muscle_ratio: meta.skeletal_muscle_ratio != null ? Number(meta.skeletal_muscle_ratio) : null,
    source: String(meta.source ?? "inbody"),
    created_at: row.created_at instanceof Date ? row.created_at.toISOString() : String(row.created_at ?? ""),
  }
}

function entryToGoal(row: Record<string, unknown>): BodyGoal {
  const meta = (row.metadata ?? {}) as Record<string, unknown>
  return {
    id: row.id as string,
    metric: String(meta.metric ?? row.title ?? ""),
    target_value: Number(meta.target_value ?? 0),
    start_value: meta.start_value != null ? Number(meta.start_value) : null,
    start_date: serializeDateOnly(meta.start_date),
    target_date: serializeDateOnly(meta.target_date),
    status: (row.status ?? "active") as BodyGoalStatus,
    created_at: row.created_at instanceof Date ? row.created_at.toISOString() : String(row.created_at ?? ""),
  }
}

export async function getBodyMeasurements(
  startDate: string | null
): Promise<BodyMeasurement[]> {
  if (startDate) {
    const rows = await sql`
      SELECT id, metadata, created_at FROM entries
      WHERE type = 'body_measurement'
        AND (metadata->>'measured_at')::timestamptz >= ${startDate}::date
      ORDER BY (metadata->>'measured_at')::timestamptz ASC
    `
    return rows.map((r) => entryToMeasurement(r as Record<string, unknown>))
  }
  const rows = await sql`
    SELECT id, metadata, created_at FROM entries
    WHERE type = 'body_measurement'
    ORDER BY (metadata->>'measured_at')::timestamptz ASC
  `
  return rows.map((r) => entryToMeasurement(r as Record<string, unknown>))
}

export async function getLatestMeasurement(): Promise<BodyMeasurement | null> {
  const rows = await sql`
    SELECT id, metadata, created_at FROM entries
    WHERE type = 'body_measurement'
    ORDER BY (metadata->>'measured_at')::timestamptz DESC
    LIMIT 1
  `
  if (!rows[0]) return null
  return entryToMeasurement(rows[0] as Record<string, unknown>)
}

export async function getBodyGoals(
  status: BodyGoalStatus | null = null
): Promise<BodyGoal[]> {
  if (status) {
    const rows = await sql`
      SELECT id, title, status, metadata, created_at FROM entries
      WHERE type = 'body_measurement_goal'
        AND status = ${status}
      ORDER BY created_at DESC
    `
    return rows.map((r) => entryToGoal(r as Record<string, unknown>))
  }
  const rows = await sql`
    SELECT id, title, status, metadata, created_at FROM entries
    WHERE type = 'body_measurement_goal'
    ORDER BY created_at DESC
  `
  return rows.map((r) => entryToGoal(r as Record<string, unknown>))
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

  const metadata = {
    metric: input.metric,
    target_value: input.target_value,
    start_value: startValue,
    start_date: startDate,
    target_date: input.target_date,
  }

  const rows = await sql`
    INSERT INTO entries (type, title, status, metadata)
    VALUES ('body_measurement_goal', ${input.metric}, 'active', ${JSON.stringify(metadata)}::jsonb)
    RETURNING id, title, status, metadata, created_at
  `
  return entryToGoal(rows[0] as Record<string, unknown>)
}

export async function updateBodyGoal(
  id: string,
  input: { target_value?: number; target_date?: string; status?: BodyGoalStatus }
): Promise<BodyGoal> {
  const updates: Record<string, unknown> = {}
  if (input.target_value !== undefined) updates.target_value = input.target_value
  if (input.target_date !== undefined) updates.target_date = input.target_date

  if (input.status !== undefined) {
    const rows = await sql`
      UPDATE entries
      SET
        status = ${input.status},
        metadata = metadata || ${JSON.stringify(updates)}::jsonb
      WHERE id = ${id}
        AND type = 'body_measurement_goal'
      RETURNING id, title, status, metadata, created_at
    `
    return entryToGoal(rows[0] as Record<string, unknown>)
  }

  const rows = await sql`
    UPDATE entries
    SET metadata = metadata || ${JSON.stringify(updates)}::jsonb
    WHERE id = ${id}
      AND type = 'body_measurement_goal'
    RETURNING id, title, status, metadata, created_at
  `
  return entryToGoal(rows[0] as Record<string, unknown>)
}

export async function getAllMeasurementsForCalibration(): Promise<
  { weight_kg: number; muscle_mass_kg: number | null; skeletal_muscle_mass_kg: number | null; body_fat_mass_kg: number | null }[]
> {
  const rows = await sql`
    SELECT
      (metadata->>'weight_kg')::numeric AS weight_kg,
      (metadata->>'muscle_mass_kg')::numeric AS muscle_mass_kg,
      (metadata->>'skeletal_muscle_mass_kg')::numeric AS skeletal_muscle_mass_kg,
      (metadata->>'body_fat_mass_kg')::numeric AS body_fat_mass_kg
    FROM entries
    WHERE type = 'body_measurement'
    ORDER BY (metadata->>'measured_at')::timestamptz ASC
  `
  return rows as { weight_kg: number; muscle_mass_kg: number | null; skeletal_muscle_mass_kg: number | null; body_fat_mass_kg: number | null }[]
}

export async function getMeasurementHistory(
  limit: number = 20,
  offset: number = 0
): Promise<{ rows: BodyMeasurement[]; total: number }> {
  const countResult = await sql`
    SELECT COUNT(*)::int AS total FROM entries
    WHERE type = 'body_measurement'
  `
  const rows = await sql`
    SELECT id, metadata, created_at FROM entries
    WHERE type = 'body_measurement'
    ORDER BY (metadata->>'measured_at')::timestamptz DESC
    LIMIT ${limit} OFFSET ${offset}
  `
  return {
    rows: rows.map((r) => entryToMeasurement(r as Record<string, unknown>)),
    total: (countResult[0] as { total: number }).total,
  }
}
