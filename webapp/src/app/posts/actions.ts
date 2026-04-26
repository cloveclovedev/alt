"use server"

import { revalidatePath } from "next/cache"
import { sql } from "@/lib/db"

export async function approveDraft(id: string, scheduledAt: string, imageUrl: string | null) {
  const metadata: Record<string, string> = { scheduled_at: scheduledAt }
  if (imageUrl) metadata.image_url = imageUrl

  await sql`
    UPDATE entries
    SET status = 'approved',
        metadata = metadata || ${JSON.stringify(metadata)}::jsonb,
        updated_at = now()
    WHERE id = ${id} AND type = 'x_draft' AND status = 'draft'
  `
  revalidatePath("/posts")
}

export async function skipDraft(id: string) {
  await sql`
    UPDATE entries
    SET status = 'skipped', updated_at = now()
    WHERE id = ${id} AND type = 'x_draft' AND status = 'draft'
  `
  revalidatePath("/posts")
}

export async function cancelApproval(id: string) {
  await sql`
    UPDATE entries
    SET status = 'draft',
        metadata = metadata - 'scheduled_at',
        updated_at = now()
    WHERE id = ${id} AND type = 'x_draft' AND status = 'approved'
  `
  revalidatePath("/posts")
}

export async function updateDraftContent(id: string, content: string) {
  await sql`
    UPDATE entries
    SET content = ${content}, updated_at = now()
    WHERE id = ${id} AND type = 'x_draft' AND status = 'draft'
  `
}

export async function updateDraftImage(id: string, imageUrl: string | null) {
  if (imageUrl) {
    await sql`
      UPDATE entries
      SET metadata = metadata || ${JSON.stringify({ image_url: imageUrl })}::jsonb,
          updated_at = now()
      WHERE id = ${id} AND type = 'x_draft' AND status = 'draft'
    `
  } else {
    await sql`
      UPDATE entries
      SET metadata = metadata - 'image_url',
          updated_at = now()
      WHERE id = ${id} AND type = 'x_draft' AND status = 'draft'
    `
  }
}
