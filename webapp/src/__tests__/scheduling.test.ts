import { describe, it, expect } from "vitest"
import { getNextAvailableSlot } from "../lib/scheduling"

describe("getNextAvailableSlot", () => {
  const defaultTimes = ["12:00", "19:00"]

  it("returns next slot today when current time is before first slot", () => {
    const now = new Date("2026-04-10T10:30:00+09:00")
    const occupied: string[] = []
    const result = getNextAvailableSlot(now, occupied, defaultTimes)
    expect(result).toBe("2026-04-10T12:00:00+09:00")
  })

  it("returns second slot when first is occupied", () => {
    const now = new Date("2026-04-10T10:30:00+09:00")
    const occupied = ["2026-04-10T12:00:00+09:00"]
    const result = getNextAvailableSlot(now, occupied, defaultTimes)
    expect(result).toBe("2026-04-10T19:00:00+09:00")
  })

  it("returns next day first slot when all today occupied", () => {
    const now = new Date("2026-04-10T10:30:00+09:00")
    const occupied = [
      "2026-04-10T12:00:00+09:00",
      "2026-04-10T19:00:00+09:00",
    ]
    const result = getNextAvailableSlot(now, occupied, defaultTimes)
    expect(result).toBe("2026-04-11T12:00:00+09:00")
  })

  it("skips past slots", () => {
    const now = new Date("2026-04-10T15:00:00+09:00")
    const occupied: string[] = []
    const result = getNextAvailableSlot(now, occupied, defaultTimes)
    expect(result).toBe("2026-04-10T19:00:00+09:00")
  })

  it("goes to next day when all remaining today are past", () => {
    const now = new Date("2026-04-10T20:00:00+09:00")
    const occupied: string[] = []
    const result = getNextAvailableSlot(now, occupied, defaultTimes)
    expect(result).toBe("2026-04-11T12:00:00+09:00")
  })
})
