"use client"

import { useState } from "react"
import { MetricChart } from "./metric-chart"
import { SECONDARY_METRICS, METRIC_CONFIG, type MetricKey } from "@/lib/body-utils"
import type { BodyMeasurement, BodyGoal } from "@/lib/types"

interface MetricSelectorProps {
  measurements: BodyMeasurement[]
  goals: BodyGoal[]
}

export function MetricSelector({ measurements, goals }: MetricSelectorProps) {
  const [selected, setSelected] = useState<MetricKey>(SECONDARY_METRICS[0])
  const goalMap = new Map(goals.map((g) => [g.metric, g]))

  return (
    <div>
      <div className="flex items-center gap-3 mb-3">
        <h2 className="text-lg font-semibold">Other Metrics</h2>
        <select
          value={selected}
          onChange={(e) => setSelected(e.target.value as MetricKey)}
          className="rounded-md border px-2 py-1 text-sm bg-background"
        >
          {SECONDARY_METRICS.map((metric) => (
            <option key={metric} value={metric}>
              {METRIC_CONFIG[metric].label}
            </option>
          ))}
        </select>
      </div>
      <MetricChart
        metric={selected}
        measurements={measurements}
        goal={goalMap.get(selected)}
      />
    </div>
  )
}
