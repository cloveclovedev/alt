import { sql } from "./db"

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
