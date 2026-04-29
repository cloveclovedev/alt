"use server"

import { revalidatePath } from "next/cache"
import { auth } from "@/lib/auth"
import {
  castValueByType,
  listConfigsWithMeta,
  setConfigValues,
} from "@/lib/config"

export interface SaveResult {
  ok: boolean
  errors?: Record<string, string>
}

export async function saveConfigValues(
  updates: Array<{ key: string; rawValue: unknown }>
): Promise<SaveResult> {
  const session = await auth()
  if (!session?.user) {
    return { ok: false, errors: { _: "Unauthenticated" } }
  }

  const all = await listConfigsWithMeta()
  const byKey = new Map(all.map((r) => [r.key, r]))

  const cast: Array<{ key: string; value: unknown }> = []
  const errors: Record<string, string> = {}

  for (const u of updates) {
    const meta = byKey.get(u.key)?.metadata
    const type = meta?.type ?? inferType(byKey.get(u.key)?.value)
    try {
      cast.push({ key: u.key, value: castValueByType(u.rawValue, type) })
    } catch (e) {
      errors[u.key] = e instanceof Error ? e.message : String(e)
    }
  }

  if (Object.keys(errors).length > 0) return { ok: false, errors }

  await setConfigValues(cast)
  revalidatePath("/config")
  return { ok: true }
}

function inferType(value: unknown): string {
  if (typeof value === "string") return "string"
  if (typeof value === "number") return "number"
  if (typeof value === "boolean") return "boolean"
  if (Array.isArray(value)) return "array"
  if (value !== null && typeof value === "object") return "object"
  return "string"
}
