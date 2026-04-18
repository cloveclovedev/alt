import { NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { getBodyMeasurements, getMeasurementHistory } from "@/lib/body"
import { periodStartDate } from "@/lib/body-utils"
import type { Period } from "@/lib/types"

const VALID_PERIODS = new Set(["30d", "90d", "6m", "1y", "all"])

export async function GET(request: Request) {
  const session = await auth()
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 })

  const { searchParams } = new URL(request.url)
  const period = searchParams.get("period") ?? "90d"
  const limit = searchParams.get("limit")
  const offset = searchParams.get("offset")

  if (!VALID_PERIODS.has(period)) {
    return NextResponse.json({ error: "Invalid period" }, { status: 400 })
  }

  // Paginated history mode
  if (limit && offset) {
    const history = await getMeasurementHistory(Number(limit), Number(offset))
    return NextResponse.json(history)
  }

  // Chart data mode
  const startDate = periodStartDate(period as Period)
  const measurements = await getBodyMeasurements(startDate)
  return NextResponse.json(measurements)
}
