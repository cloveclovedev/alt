import Link from "next/link"
import { Card, CardContent, CardHeader, CardTitle, CardAction } from "@/components/ui/card"
import { getBodyMeasurements, getBodyGoals, getLatestMeasurement, getAllMeasurementsForCalibration } from "@/lib/body"
import { getConfig } from "@/lib/config"
import { periodStartDate, PRIMARY_METRICS, computeCalibration, derivedGoals } from "@/lib/body-utils"
import { MetricChart } from "./metric-chart"
import type { BodyGoal } from "@/lib/types"

export async function BodySummaryCard() {
  const startDate = periodStartDate("90d")
  const [measurements, goals, latest, allForCalibration, heightM] = await Promise.all([
    getBodyMeasurements(startDate),
    getBodyGoals("active"),
    getLatestMeasurement(),
    getAllMeasurementsForCalibration(),
    getConfig<number>("body.height_m", 1.7),
  ])

  if (measurements.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Body Composition</CardTitle>
          <CardAction>
            <Link href="/body" className="text-sm text-muted-foreground hover:text-foreground">
              View details →
            </Link>
          </CardAction>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">No measurements yet</p>
        </CardContent>
      </Card>
    )
  }

  const goalMap = new Map(goals.map((g) => [g.metric, g]))

  // Compute derived goals
  const calibration = computeCalibration(allForCalibration)
  const weightGoal = goalMap.get("weight_kg")
  const skmGoal = goalMap.get("skeletal_muscle_mass_kg")

  if (calibration && weightGoal && skmGoal) {
    const derived = derivedGoals({
      targetWeight: weightGoal.target_value,
      targetSkm: skmGoal.target_value,
      heightM: heightM ?? 1.7,
      calibration,
    })

    const baseDate = weightGoal.target_date > skmGoal.target_date
      ? weightGoal.target_date : skmGoal.target_date
    const baseStartDate = weightGoal.start_date < skmGoal.start_date
      ? weightGoal.start_date : skmGoal.start_date

    if (!goalMap.has("body_fat_percent")) {
      goalMap.set("body_fat_percent", {
        id: "derived-bf", metric: "body_fat_percent",
        target_value: derived.bodyFatPercent,
        start_value: latest?.body_fat_percent != null ? Number(latest.body_fat_percent) : null,
        start_date: baseStartDate, target_date: baseDate,
        status: "active", created_at: "",
      } as BodyGoal)
    }
    if (!goalMap.has("ffmi")) {
      goalMap.set("ffmi", {
        id: "derived-ffmi", metric: "ffmi",
        target_value: derived.ffmi,
        start_value: latest?.ffmi != null ? Number(latest.ffmi) : null,
        start_date: baseStartDate, target_date: baseDate,
        status: "active", created_at: "",
      } as BodyGoal)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Body Composition</CardTitle>
        <CardAction>
          <Link href="/body" className="text-sm text-muted-foreground hover:text-foreground">
            View details →
          </Link>
        </CardAction>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-3">
          {PRIMARY_METRICS.map((metric) => (
            <MetricChart
              key={metric}
              metric={metric}
              measurements={measurements}
              goal={goalMap.get(metric)}
              compact
            />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
