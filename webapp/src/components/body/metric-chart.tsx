"use client"

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts"
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart"
import type { ChartConfig } from "@/components/ui/chart"
import type { BodyMeasurement, BodyGoal } from "@/lib/types"
import { METRIC_CONFIG, computeIdealLine, type MetricKey } from "@/lib/body-utils"

interface MetricChartProps {
  metric: MetricKey
  measurements: BodyMeasurement[]
  goal?: BodyGoal | null
  compact?: boolean
}

export function MetricChart({ metric, measurements, goal, compact = false }: MetricChartProps) {
  const config = METRIC_CONFIG[metric]
  const idealPoints = goal ? computeIdealLine(goal) : []

  const chartData = measurements.map((m) => ({
    timestamp: new Date(m.measured_at).getTime(),
    value: m[metric] as number | null,
  })).filter((d) => d.value !== null)

  const idealData = idealPoints.map((p) => ({
    timestamp: new Date(p.date).getTime(),
    ideal: p.value,
  }))

  const mergedData = mergeChartData(chartData, idealData)

  const latestValue = chartData.length > 0 ? chartData[chartData.length - 1].value : null

  const chartConfig: ChartConfig = {
    value: { label: config.label, color: config.color },
    ideal: { label: "Goal", color: config.color },
  }

  if (compact) {
    return (
      <div className="rounded-lg border p-3">
        <div className="flex items-center justify-between mb-1">
          <span className="text-xs font-semibold" style={{ color: config.color }}>
            {config.label}
          </span>
          <span className="text-sm font-bold">
            {latestValue != null ? `${latestValue}${config.unit}` : "—"}
          </span>
        </div>
        <ChartContainer config={chartConfig} className="h-[80px] w-full">
          <LineChart data={mergedData} margin={{ top: 4, right: 4, bottom: 0, left: 4 }}>
            <XAxis dataKey="timestamp" type="number" scale="time" domain={["dataMin", "dataMax"]} hide />
            <YAxis domain={["auto", "auto"]} hide />
            <Line
              type="monotone"
              dataKey="value"
              stroke={config.color}
              strokeWidth={2}
              dot={false}
              connectNulls
            />
            {idealData.length > 0 && (
              <Line
                type="monotone"
                dataKey="ideal"
                stroke={config.color}
                strokeWidth={1.5}
                strokeDasharray="4 3"
                strokeOpacity={0.4}
                dot={false}
                connectNulls
              />
            )}
          </LineChart>
        </ChartContainer>
      </div>
    )
  }

  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-semibold" style={{ color: config.color }}>
          {config.label}
        </span>
        <span className="text-lg font-bold">
          {latestValue != null ? `${latestValue}${config.unit ? ` ${config.unit}` : ""}` : "—"}
        </span>
      </div>
      <ChartContainer config={chartConfig} className="h-[200px] w-full">
        <LineChart data={mergedData} margin={{ top: 8, right: 16, bottom: 0, left: 8 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis
            dataKey="timestamp"
            type="number"
            scale="time"
            domain={["dataMin", "dataMax"]}
            tickFormatter={(ts: number) => {
              const date = new Date(ts)
              return `${date.getMonth() + 1}/${date.getDate()}`
            }}
            className="text-xs"
          />
          <YAxis domain={["auto", "auto"]} className="text-xs" />
          <ChartTooltip
            content={
              <ChartTooltipContent
                labelFormatter={(label) => {
                  if (typeof label === "number") return new Date(label).toLocaleDateString()
                  if (typeof label === "string") return new Date(label).toLocaleDateString()
                  return String(label)
                }}
              />
            }
          />
          <Line
            type="monotone"
            dataKey="value"
            stroke={config.color}
            strokeWidth={2}
            dot={{ r: 3, fill: config.color }}
            connectNulls
          />
          {idealData.length > 0 && (
            <Line
              type="monotone"
              dataKey="ideal"
              stroke={config.color}
              strokeWidth={1.5}
              strokeDasharray="4 3"
              strokeOpacity={0.4}
              dot={false}
              connectNulls
            />
          )}
        </LineChart>
      </ChartContainer>
      {goal && (
        <div className="mt-1 text-xs text-muted-foreground text-right">
          Goal: {goal.target_value}{config.unit ? ` ${config.unit}` : ""} by {goal.target_date}
        </div>
      )}
    </div>
  )
}

function mergeChartData(
  actual: { timestamp: number; value: number | null }[],
  ideal: { timestamp: number; ideal: number }[]
): { timestamp: number; value?: number | null; ideal?: number }[] {
  const map = new Map<number, { value?: number | null; ideal?: number }>()

  for (const a of actual) {
    map.set(a.timestamp, { ...map.get(a.timestamp), value: a.value })
  }
  for (const i of ideal) {
    map.set(i.timestamp, { ...map.get(i.timestamp), ideal: i.ideal })
  }

  return Array.from(map.entries())
    .map(([timestamp, data]) => ({ timestamp, ...data }))
    .sort((a, b) => a.timestamp - b.timestamp)
}
