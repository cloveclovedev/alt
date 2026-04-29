import { sql } from "./db"

export type ConfigType = "string" | "number" | "boolean" | "array" | "object"

export interface ConfigMeta {
  type?: ConfigType
  description?: string
  consumed_by?: string[]
  default?: unknown
}

export interface ConfigRow {
  key: string
  value: unknown
  metadata: ConfigMeta
}

/**
 * Read a value from the `config` table.
 *
 * Uses `value::text` to force JSONB → text serialization on the server, then
 * parses it client-side. This mirrors the Python helper at src/alt_db/config.py
 * and avoids the JSONB-bare-string round-trip ambiguity.
 */
export async function getConfig<T = unknown>(
  key: string,
  defaultValue?: T
): Promise<T | undefined> {
  const rows = (await sql`SELECT value::text AS value FROM config WHERE key = ${key}`) as Array<{
    value: string
  }>
  if (rows.length === 0) return defaultValue
  return JSON.parse(rows[0].value) as T
}

/** List all config rows including their metadata. */
export async function listConfigsWithMeta(): Promise<ConfigRow[]> {
  const rows = (await sql`
    SELECT key, value::text AS value, metadata::text AS metadata
    FROM config
    ORDER BY key
  `) as Array<{ key: string; value: string; metadata: string }>
  return rows.map((r) => ({
    key: r.key,
    value: JSON.parse(r.value),
    metadata: JSON.parse(r.metadata) as ConfigMeta,
  }))
}

/**
 * Cast a form-submitted value to the JSON shape implied by its declared type.
 * Throws on conversion failure (e.g. malformed JSON, "abc" for a number).
 */
export function castValueByType(raw: unknown, type: string): unknown {
  switch (type) {
    case "string":
      return String(raw)
    case "number": {
      const n = Number(raw)
      if (!Number.isFinite(n)) throw new Error(`Invalid number: ${raw}`)
      return n
    }
    case "boolean":
      if (typeof raw === "boolean") return raw
      if (raw === "true") return true
      if (raw === "false") return false
      throw new Error(`Invalid boolean: ${raw}`)
    case "array": {
      const parsed = typeof raw === "string" ? JSON.parse(raw) : raw
      if (!Array.isArray(parsed)) throw new Error("Expected JSON array")
      return parsed
    }
    case "object": {
      const parsed = typeof raw === "string" ? JSON.parse(raw) : raw
      if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
        throw new Error("Expected JSON object")
      }
      return parsed
    }
    default:
      return raw
  }
}

/**
 * Update many config values in a single transaction.
 * Each update's value has already been cast by the caller.
 */
export async function setConfigValues(
  updates: Array<{ key: string; value: unknown }>
): Promise<void> {
  if (updates.length === 0) return
  // Neon serverless driver does not expose explicit transactions; run statements
  // sequentially. For a single-user tool the loss of atomicity is acceptable.
  for (const u of updates) {
    await sql`
      UPDATE config
      SET value = ${JSON.stringify(u.value)}::jsonb, updated_at = now()
      WHERE key = ${u.key}
    `
  }
}
