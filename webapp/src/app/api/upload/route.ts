import { put } from "@vercel/blob"
import { auth } from "@/lib/auth"
import { NextResponse } from "next/server"

export async function POST(request: Request) {
  const session = await auth()
  if (!session?.user) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 })
  }

  const formData = await request.formData()
  const file = formData.get("file") as File | null
  if (!file) {
    return NextResponse.json({ error: "No file provided" }, { status: 400 })
  }

  const blob = await put(`x-posts/${Date.now()}-${file.name}`, file, {
    access: "public",
  })

  return NextResponse.json({ url: blob.url })
}
