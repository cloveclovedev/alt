import { METRIC_CONFIG, type MetricKey } from "@/lib/body-utils"
import type { BodyMeasurement } from "@/lib/types"

interface LatestSummaryProps {
  measurement: BodyMeasurement | null
}

const ALL_METRICS: MetricKey[] = [
  "weight_kg", "body_fat_percent", "muscle_mass_kg", "ffmi",
  "bmi", "skeletal_muscle_mass_kg", "body_fat_mass_kg",
  "basal_metabolic_rate", "inbody_score", "waist_hip_ratio",
  "visceral_fat_level", "skeletal_muscle_ratio",
]

export function LatestSummary({ measurement }: LatestSummaryProps) {
  if (!measurement) {
    return <p className="text-muted-foreground">No measurements yet</p>
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-lg font-semibold">Latest Measurement</h2>
        <span className="text-sm text-muted-foreground">
          {new Date(measurement.measured_at).toLocaleDateString()}
        </span>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {ALL_METRICS.map((metric) => {
          const value = measurement[metric] as number | null
          const config = METRIC_CONFIG[metric]
          return (
            <div key={metric} className="rounded-lg border p-3">
              <div className="text-xs text-muted-foreground">{config.label}</div>
              <div className="text-lg font-bold">
                {value != null ? `${value}${config.unit ? ` ${config.unit}` : ""}` : "—"}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
