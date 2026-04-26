import { sql } from "./db"
import type { Entry, EntryFilters } from "./types"

export async function getActiveGoals(): Promise<Entry[]> {
  const rows = await sql`
    SELECT id, type, title, content, status, metadata, parent_id, created_at, updated_at FROM entries
    WHERE type = 'goal' AND status = 'active'
    ORDER BY created_at DESC
  `
  return rows as Entry[]
}

export async function getRecentMemos(days: number): Promise<Entry[]> {
  const rows = await sql`
    SELECT id, type, title, content, status, metadata, parent_id, created_at, updated_at FROM entries
    WHERE type = 'memo'
      AND created_at >= NOW() - make_interval(days => ${days})
    ORDER BY created_at DESC
  `
  return rows as Entry[]
}

export async function getUpcomingDeadlines(days: number): Promise<Entry[]> {
  const rows = await sql`
    SELECT id, type, title, content, status, metadata, parent_id, created_at, updated_at FROM entries
    WHERE type = 'goal'
      AND status = 'active'
      AND metadata->>'target_date' ~ '^\d{4}-\d{2}-\d{2}$'
      AND (metadata->>'target_date')::date >= CURRENT_DATE
      AND (metadata->>'target_date')::date <= CURRENT_DATE + make_interval(days => ${days})
    ORDER BY (metadata->>'target_date')::date ASC
  `
  return rows as Entry[]
}

export async function listEntries(filters: EntryFilters): Promise<Entry[]> {
  const { type = null, status = null, search = null, limit = 50 } = filters
  const searchPattern = search ? `%${search}%` : null

  const rows = await sql`
    SELECT id, type, title, content, status, metadata, parent_id, created_at, updated_at FROM entries
    WHERE (${type}::text IS NULL OR type = ${type})
      AND (${status}::text IS NULL OR status = ${status})
      AND (${searchPattern}::text IS NULL OR title ILIKE ${searchPattern} OR content ILIKE ${searchPattern})
    ORDER BY created_at DESC
    LIMIT ${limit}
  `
  return rows as Entry[]
}

export async function getLatestRoutineEntries(): Promise<Entry[]> {
  const rows = await sql`
    SELECT DISTINCT ON (title) id, type, title, content, status, metadata, parent_id, created_at, updated_at
    FROM entries
    WHERE type = 'routine_event'
    ORDER BY title, created_at DESC
  `
  return rows as Entry[]
}

export async function getXDrafts(): Promise<Entry[]> {
  const rows = await sql`
    SELECT * FROM entries
    WHERE type = 'x_draft'
    ORDER BY
      CASE status
        WHEN 'draft' THEN 0
        WHEN 'approved' THEN 1
        WHEN 'posted' THEN 2
        WHEN 'skipped' THEN 3
        ELSE 99
      END,
      created_at DESC
  `
  return rows as Entry[]
}

export async function getOccupiedSlots(): Promise<string[]> {
  const rows = await sql`
    SELECT metadata->>'scheduled_at' as scheduled_at
    FROM entries
    WHERE type = 'x_draft'
      AND status IN ('approved', 'posted')
      AND metadata->>'scheduled_at' IS NOT NULL
  `
  return rows.map((r) => r.scheduled_at as string)
}
