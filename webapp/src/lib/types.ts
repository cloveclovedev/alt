export interface Entry {
  id: string
  type: string
  title: string
  content: string | null
  status: string | null
  tags: string[]
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface RoutineEvent {
  id: string
  routine_name: string
  category: string
  completed_at: string
  kind: string
  note: string | null
}

export interface EntryFilters {
  type?: string | null
  status?: string | null
  tag?: string | null
  search?: string | null
  limit?: number
}

export interface BodyMeasurement {
  id: string
  measured_at: string
  weight_kg: number
  skeletal_muscle_mass_kg: number | null
  muscle_mass_kg: number | null
  body_fat_mass_kg: number | null
  body_fat_percent: number | null
  bmi: number | null
  basal_metabolic_rate: number | null
  inbody_score: number | null
  waist_hip_ratio: number | null
  visceral_fat_level: number | null
  ffmi: number | null
  skeletal_muscle_ratio: number | null
  source: string
  created_at: string
}

export type BodyGoalStatus = "active" | "achieved" | "expired" | "cancelled"

export interface BodyGoal {
  id: string
  metric: string
  target_value: number
  start_value: number | null
  start_date: string
  target_date: string
  status: BodyGoalStatus
  created_at: string
}

export type Period = "30d" | "90d" | "6m" | "1y" | "all"
