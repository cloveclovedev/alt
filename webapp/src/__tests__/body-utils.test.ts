import { describe, it, expect } from "vitest"
import {
  periodToDays,
  periodStartDate,
  computeIdealLine,
  computeCalibration,
  derivedGoals,
  METRIC_CONFIG,
  type MetricKey,
} from "@/lib/body-utils"

describe("periodToDays", () => {
  it("converts period strings to day counts", () => {
    expect(periodToDays("30d")).toBe(30)
    expect(periodToDays("90d")).toBe(90)
    expect(periodToDays("6m")).toBe(180)
    expect(periodToDays("1y")).toBe(365)
    expect(periodToDays("all")).toBeNull()
  })
})

describe("periodStartDate", () => {
  it("returns ISO date string for bounded periods", () => {
    const now = new Date("2026-04-09T00:00:00Z")
    expect(periodStartDate("30d", now)).toBe("2026-03-10")
    expect(periodStartDate("90d", now)).toBe("2026-01-09")
  })

  it("returns null for 'all'", () => {
    expect(periodStartDate("all")).toBeNull()
  })
})

describe("computeIdealLine", () => {
  it("returns two points for linear interpolation", () => {
    const goal = {
      start_value: 75,
      target_value: 70,
      start_date: "2026-01-01",
      target_date: "2026-07-01",
    }
    const points = computeIdealLine(goal)
    expect(points).toEqual([
      { date: "2026-01-01", value: 75 },
      { date: "2026-07-01", value: 70 },
    ])
  })

  it("returns empty array when start_value is null", () => {
    const goal = {
      start_value: null,
      target_value: 70,
      start_date: "2026-01-01",
      target_date: "2026-07-01",
    }
    expect(computeIdealLine(goal)).toEqual([])
  })
})

describe("computeCalibration", () => {
  it("computes correction and skmRatio from measurements", () => {
    const measurements = [
      { weight_kg: 65, muscle_mass_kg: 47, skeletal_muscle_mass_kg: 28, body_fat_mass_kg: 15.3 },
      { weight_kg: 66, muscle_mass_kg: 48, skeletal_muscle_mass_kg: 28.5, body_fat_mass_kg: 15.2 },
    ]
    const result = computeCalibration(measurements)
    expect(result).not.toBeNull()
    // correction = avg((65-47-15.3), (66-48-15.2)) = avg(2.7, 2.8) = 2.75
    expect(result!.correction).toBeCloseTo(2.75, 2)
    // skmRatio = avg(28/47, 28.5/48) = avg(0.5957, 0.59375) = 0.5947
    expect(result!.skmRatio).toBeCloseTo(0.5947, 3)
  })

  it("returns null when no valid measurements", () => {
    const measurements = [
      { weight_kg: 65, muscle_mass_kg: null, skeletal_muscle_mass_kg: null, body_fat_mass_kg: null },
    ]
    expect(computeCalibration(measurements)).toBeNull()
  })
})

describe("derivedGoals", () => {
  it("estimates body fat % and FFMI from weight + skeletal muscle targets", () => {
    const result = derivedGoals({
      targetWeight: 65,
      targetSkm: 30,
      heightM: 1.70,
      calibration: { correction: 2.71, skmRatio: 0.597 },
    })
    // mm_est = 30 / 0.597 = 50.25
    // bf_mass = 65 - 50.25 - 2.71 = 12.04
    // bf% = 12.04 / 65 * 100 = 18.5%
    expect(result.bodyFatPercent).toBeCloseTo(18.5, 0)
    // lbm = 65 - 12.04 = 52.96
    // ffmi = 52.96 / 1.70^2 + 6.1 * (1.8 - 1.70) = 18.33 + 0.61 = 18.94
    expect(result.ffmi).toBeCloseTo(18.9, 0)
  })
})

describe("METRIC_CONFIG", () => {
  it("has config for primary metrics", () => {
    const keys: MetricKey[] = ["weight_kg", "body_fat_percent", "muscle_mass_kg", "ffmi"]
    for (const key of keys) {
      expect(METRIC_CONFIG[key]).toBeDefined()
      expect(METRIC_CONFIG[key].label).toBeTruthy()
      expect(METRIC_CONFIG[key].color).toBeTruthy()
      expect(METRIC_CONFIG[key].unit).toBeDefined()
    }
  })
})
