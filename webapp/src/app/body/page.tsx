import { getBodyMeasurements, getBodyGoals, getLatestMeasurement, getAllMeasurementsForCalibration } from "@/lib/body"
import { getConfig } from "@/lib/config"
import {
  periodStartDate,
  PRIMARY_METRICS,
  computeCalibration,
  derivedGoals,
} from "@/lib/body-utils"
import { MetricChart } from "@/components/body/metric-chart"
import { PeriodSelector } from "@/components/body/period-selector"
import { MetricSelector } from "@/components/body/metric-selector"
import { CompositionGoalCard } from "@/components/body/goal-card"
import { LatestSummary } from "@/components/body/latest-summary"
import { MeasurementHistory } from "@/components/body/measurement-history"
import type { Period, BodyGoal } from "@/lib/types"

interface BodyPageProps {
  searchParams: Promise<{ period?: string }>
}

export default async function BodyPage({ searchParams }: BodyPageProps) {
  const { period: periodParam } = await searchParams
  const period = (periodParam ?? "90d") as Period
  const startDate = periodStartDate(period)

  const [measurements, allGoals, latest, allForCalibration, heightM] = await Promise.all([
    getBodyMeasurements(startDate),
    getBodyGoals(),
    getLatestMeasurement(),
    getAllMeasurementsForCalibration(),
    getConfig<number>("body.height_m", 1.7),
  ])

  const activeGoals = allGoals.filter((g) => g.status === "active")
  const goalMap = new Map(activeGoals.map((g) => [g.metric, g]))

  // Compute derived goals from weight + skeletal muscle mass targets
  const calibration = computeCalibration(allForCalibration)
  const weightGoal = goalMap.get("weight_kg")
  const skmGoal = goalMap.get("skeletal_muscle_mass_kg")

  let derivedGoalMap = new Map<string, BodyGoal>()
  let derived: { bodyFatPercent: number; ffmi: number } | null = null

  if (calibration && weightGoal && skmGoal) {
    derived = derivedGoals({
      targetWeight: weightGoal.target_value,
      targetSkm: skmGoal.target_value,
      heightM: heightM ?? 1.7,
      calibration,
    })

    // Create synthetic goal objects for charts
    const baseDate = weightGoal.target_date > skmGoal.target_date
      ? weightGoal.target_date : skmGoal.target_date
    const baseStartDate = weightGoal.start_date < skmGoal.start_date
      ? weightGoal.start_date : skmGoal.start_date

    const latestBfPct = latest?.body_fat_percent ?? null
    const latestFfmi = latest?.ffmi ?? null

    derivedGoalMap.set("body_fat_percent", {
      id: "derived-bf",
      metric: "body_fat_percent",
      target_value: derived.bodyFatPercent,
      start_value: latestBfPct != null ? Number(latestBfPct) : null,
      start_date: baseStartDate,
      target_date: baseDate,
      status: "active",
      created_at: "",
    })
    derivedGoalMap.set("ffmi", {
      id: "derived-ffmi",
      metric: "ffmi",
      target_value: derived.ffmi,
      start_value: latestFfmi != null ? Number(latestFfmi) : null,
      start_date: baseStartDate,
      target_date: baseDate,
      status: "active",
      created_at: "",
    })
  }

  // Merge real goals + derived goals for chart display
  const chartGoalMap = new Map([...goalMap, ...derivedGoalMap])

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="flex items-center justify-between mb-8">
        <h1 className="text-3xl font-bold">Body Composition</h1>
        <PeriodSelector />
      </div>

      <section className="mb-8">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {PRIMARY_METRICS.map((metric) => (
            <MetricChart
              key={metric}
              metric={metric}
              measurements={measurements}
              goal={chartGoalMap.get(metric)}
            />
          ))}
        </div>
      </section>

      <section className="mb-8">
        <MetricSelector measurements={measurements} goals={activeGoals} />
      </section>

      <section className="mb-8">
        <h2 className="text-lg font-semibold mb-3">Goals</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <CompositionGoalCard
            weightGoal={weightGoal}
            skmGoal={skmGoal}
            derivedBfPct={derived?.bodyFatPercent ?? null}
            derivedFfmi={derived?.ffmi ?? null}
            pastGoals={allGoals.filter(
              (g) => (g.metric === "weight_kg" || g.metric === "skeletal_muscle_mass_kg") && g.status !== "active"
            )}
          />
        </div>
      </section>

      <section className="mb-8">
        <LatestSummary measurement={latest} />
      </section>

      <section className="mb-8">
        <MeasurementHistory />
      </section>
    </div>
  )
}
