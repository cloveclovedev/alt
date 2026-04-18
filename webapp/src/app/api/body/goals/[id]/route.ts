import { NextResponse } from "next/server"
import { auth } from "@/lib/auth"
import { updateBodyGoal } from "@/lib/body"

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const session = await auth()
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 })

  const { id } = await params
  const body = await request.json()
  const goal = await updateBodyGoal(id, body)
  return NextResponse.json(goal)
}
