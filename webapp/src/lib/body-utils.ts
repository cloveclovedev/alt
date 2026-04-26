import type { Period } from "./types"

export type MetricKey =
  | "weight_kg"
  | "skeletal_muscle_mass_kg"
  | "muscle_mass_kg"
  | "body_fat_mass_kg"
  | "body_fat_percent"
  | "bmi"
  | "basal_metabolic_rate"
  | "inbody_score"
  | "waist_hip_ratio"
  | "visceral_fat_level"
  | "ffmi"
  | "skeletal_muscle_ratio"

export interface MetricConfig {
  label: string
  color: string
  unit: string
}

export const METRIC_CONFIG: Record<MetricKey, MetricConfig> = {
  weight_kg: { label: "Weight", color: "#60a5fa", unit: "kg" },
  body_fat_percent: { label: "Body Fat %", color: "#f472b6", unit: "%" },
  muscle_mass_kg: { label: "Muscle Mass", color: "#34d399", unit: "kg" },
  ffmi: { label: "FFMI", color: "#a78bfa", unit: "" },
  skeletal_muscle_mass_kg: { label: "Skeletal Muscle", color: "#38bdf8", unit: "kg" },
  body_fat_mass_kg: { label: "Body Fat Mass", color: "#fb923c", unit: "kg" },
  bmi: { label: "BMI", color: "#facc15", unit: "" },
  basal_metabolic_rate: { label: "BMR", color: "#f87171", unit: "kcal" },
  inbody_score: { label: "InBody Score", color: "#2dd4bf", unit: "" },
  waist_hip_ratio: { label: "Waist-Hip Ratio", color: "#c084fc", unit: "" },
  visceral_fat_level: { label: "Visceral Fat", color: "#fb7185", unit: "" },
  skeletal_muscle_ratio: { label: "Skeletal Muscle %", color: "#4ade80", unit: "%" },
}

export const PRIMARY_METRICS: MetricKey[] = [
  "weight_kg",
  "body_fat_percent",
  "skeletal_muscle_mass_kg",
  "ffmi",
]

export const SECONDARY_METRICS: MetricKey[] = [
  "bmi",
  "inbody_score",
  "basal_metabolic_rate",
  "muscle_mass_kg",
  "waist_hip_ratio",
  "visceral_fat_level",
]

export function periodToDays(period: Period): number | null {
  switch (period) {
    case "30d": return 30
    case "90d": return 90
    case "6m": return 180
    case "1y": return 365
    case "all": return null
  }
}

export function periodStartDate(period: Period, now: Date = new Date()): string | null {
  const days = periodToDays(period)
  if (days === null) return null
  const start = new Date(now)
  start.setDate(start.getDate() - days)
  return start.toISOString().slice(0, 10)
}

interface IdealLineInput {
  start_value: number | null
  target_value: number
  start_date: string
  target_date: string
}

export interface IdealPoint {
  date: string
  value: number
}

export function computeIdealLine(goal: IdealLineInput): IdealPoint[] {
  if (goal.start_value === null) return []
  return [
    { date: goal.start_date, value: goal.start_value },
    { date: goal.target_date, value: goal.target_value },
  ]
}

// --- Derived goal estimation ---
// Body fat % and FFMI can be estimated from weight + skeletal muscle mass
// using two calibration constants derived from historical InBody data:
//   correction = avg(weight - muscle_mass - body_fat_mass)  ~2.7kg (bone/organs/water)
//   skmRatio   = avg(skeletal_muscle_mass / muscle_mass)    ~0.597

export interface CalibrationConstants {
  correction: number  // non-muscle, non-fat mass (kg)
  skmRatio: number    // skeletal_muscle_mass / muscle_mass ratio
}

export interface DerivedGoalInput {
  targetWeight: number
  targetSkm: number
  heightM: number
  calibration: CalibrationConstants
}

export interface DerivedGoals {
  bodyFatPercent: number
  ffmi: number
}

export function computeCalibration(
  measurements: { weight_kg: number; muscle_mass_kg: number | null; skeletal_muscle_mass_kg: number | null; body_fat_mass_kg: number | null }[]
): CalibrationConstants | null {
  const valid = measurements.filter(
    (m) => m.muscle_mass_kg != null && m.skeletal_muscle_mass_kg != null && m.body_fat_mass_kg != null
  ) as { weight_kg: number; muscle_mass_kg: number; skeletal_muscle_mass_kg: number; body_fat_mass_kg: number }[]

  if (valid.length === 0) return null

  const corrections = valid.map((m) => m.weight_kg - m.muscle_mass_kg - m.body_fat_mass_kg)
  const ratios = valid.map((m) => m.skeletal_muscle_mass_kg / m.muscle_mass_kg)

  return {
    correction: corrections.reduce((a, b) => a + b, 0) / corrections.length,
    skmRatio: ratios.reduce((a, b) => a + b, 0) / ratios.length,
  }
}

export function derivedGoals(input: DerivedGoalInput): DerivedGoals {
  const { targetWeight, targetSkm, heightM, calibration } = input
  const muscleMassEst = targetSkm / calibration.skmRatio
  const bodyFatMassEst = targetWeight - muscleMassEst - calibration.correction
  const bodyFatPercent = (bodyFatMassEst / targetWeight) * 100
  const lbm = targetWeight - bodyFatMassEst
  const ffmi = lbm / (heightM * heightM) + 6.1 * (1.8 - heightM)
  return {
    bodyFatPercent: Math.round(bodyFatPercent * 10) / 10,
    ffmi: Math.round(ffmi * 100) / 100,
  }
}

/** Metrics that users set directly (weight, skeletal muscle). */
export const SETTABLE_METRICS: MetricKey[] = ["weight_kg", "skeletal_muscle_mass_kg"]

/** Metrics derived from settable goals (body fat %, FFMI). */
export const DERIVED_METRICS: MetricKey[] = ["body_fat_percent", "ffmi"]
