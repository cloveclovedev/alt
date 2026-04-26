import { neon } from "@neondatabase/serverless"
import type { NeonQueryFunction } from "@neondatabase/serverless"

let _sql: NeonQueryFunction<false, false> | undefined

function getSql(): NeonQueryFunction<false, false> {
  if (!_sql) {
    const databaseUrl = process.env.DATABASE_URL
    if (!databaseUrl) throw new Error("DATABASE_URL is not set")
    _sql = neon(databaseUrl)
  }
  return _sql
}

// Lazy proxy: defers neon() initialization until first query
export const sql = new Proxy(
  function () {} as unknown as NeonQueryFunction<false, false>,
  {
    apply(_target, _thisArg, args) {
      const fn = getSql()
      return (fn as (...a: unknown[]) => unknown)(...args)
    },
    get(_target, prop) {
      const fn = getSql()
      const value = fn[prop as keyof typeof fn]
      return typeof value === "function" ? value.bind(fn) : value
    },
  }
)
