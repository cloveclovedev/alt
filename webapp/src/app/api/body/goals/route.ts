import { NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { getBodyGoals, createBodyGoal } from "@/lib/body"
import type { BodyGoalStatus } from "@/lib/types"

const VALID_STATUSES = new Set(["active", "achieved", "expired", "cancelled"])

export async function GET(request: Request) {
  const session = await auth()
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 })

  const { searchParams } = new URL(request.url)
  const status = searchParams.get("status")

  if (status && !VALID_STATUSES.has(status)) {
    return NextResponse.json({ error: "Invalid status" }, { status: 400 })
  }

  const goals = await getBodyGoals(status as BodyGoalStatus | null)
  return NextResponse.json(goals)
}

export async function POST(request: Request) {
  const session = await auth()
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 })

  const body = await request.json()
  const { metric, target_value, target_date } = body

  if (!metric || target_value == null || !target_date) {
    return NextResponse.json({ error: "Missing required fields" }, { status: 400 })
  }

  const goal = await createBodyGoal({ metric, target_value, target_date })
  return NextResponse.json(goal, { status: 201 })
}
