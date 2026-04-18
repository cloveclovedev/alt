/**
 * Find the next available posting slot given default times and occupied slots.
 * All times are in JST (+09:00).
 */
export function getNextAvailableSlot(
  now: Date,
  occupiedSlots: string[],
  defaultTimes: string[],
): string {
  const occupiedSet = new Set(occupiedSlots)
  const jstOffset = 9 * 60 // minutes

  // Get current date in JST
  const jstNow = new Date(now.getTime() + (jstOffset + now.getTimezoneOffset()) * 60000)

  // Search up to 30 days ahead
  for (let dayOffset = 0; dayOffset < 30; dayOffset++) {
    const date = new Date(jstNow)
    date.setDate(date.getDate() + dayOffset)
    const dateStr = formatDateJST(date)

    for (const time of defaultTimes) {
      const slotISO = `${dateStr}T${time}:00+09:00`
      const slotDate = new Date(slotISO)

      if (slotDate <= now) continue
      if (occupiedSet.has(slotISO)) continue

      return slotISO
    }
  }

  throw new Error("No available posting slot found within 30 days")
}

function formatDateJST(date: Date): string {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, "0")
  const d = String(date.getDate()).padStart(2, "0")
  return `${y}-${m}-${d}`
}
