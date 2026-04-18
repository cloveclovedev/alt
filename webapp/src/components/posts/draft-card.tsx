"use client"

import { useState, useTransition } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { approveDraft, skipDraft, updateDraftContent, updateDraftImage } from "@/app/posts/actions"
import type { Entry } from "@/lib/types"

const MAX_CHARS = 280

interface DraftCardProps {
  draft: Entry
  nextSlot: string
  defaultTimes: string[]
  occupiedSlots: string[]
}

export function DraftCard({ draft, nextSlot, defaultTimes, occupiedSlots }: DraftCardProps) {
  const meta = draft.metadata as {
    image_url?: string
    post_type?: string
    reply_link?: string
  }

  const [content, setContent] = useState(draft.content ?? "")
  const [imageUrl, setImageUrl] = useState<string | null>(meta.image_url ?? null)
  const [scheduledAt, setScheduledAt] = useState(nextSlot)
  const [useCustomTime, setUseCustomTime] = useState(false)
  const [isPending, startTransition] = useTransition()
  const [contentDirty, setContentDirty] = useState(false)

  const charCount = content.length
  const isOverLimit = charCount > MAX_CHARS

  function handleContentChange(value: string) {
    setContent(value)
    setContentDirty(true)
  }

  function handleContentBlur() {
    if (contentDirty) {
      startTransition(async () => {
        await updateDraftContent(draft.id, content)
        setContentDirty(false)
      })
    }
  }

  async function handleImageUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return

    const formData = new FormData()
    formData.append("file", file)

    const res = await fetch("/api/upload", { method: "POST", body: formData })
    const data = await res.json()
    if (data.url) {
      setImageUrl(data.url)
      startTransition(() => updateDraftImage(draft.id, data.url))
    }
  }

  function handleRemoveImage() {
    setImageUrl(null)
    startTransition(() => updateDraftImage(draft.id, null))
  }

  function handleApprove() {
    if (isOverLimit) return
    startTransition(() => approveDraft(draft.id, scheduledAt, imageUrl))
  }

  function handleSkip() {
    startTransition(() => skipDraft(draft.id))
  }

  // Format scheduled_at for datetime-local input (JST)
  function toDatetimeLocal(iso: string): string {
    // "2026-04-10T12:00:00+09:00" -> "2026-04-10T12:00"
    return iso.slice(0, 16)
  }

  function fromDatetimeLocal(local: string): string {
    return `${local}:00+09:00`
  }

  return (
    <Card>
      <CardContent className="space-y-3 pt-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Badge variant="outline">Draft</Badge>
            {meta.post_type && (
              <Badge variant="secondary" className="text-xs">
                {meta.post_type}
              </Badge>
            )}
          </div>
          <span className="text-xs text-muted-foreground">
            {new Date(draft.created_at).toLocaleString("ja-JP", { timeZone: "Asia/Tokyo" })}
          </span>
        </div>

        <div>
          <textarea
            value={content}
            onChange={(e) => handleContentChange(e.target.value)}
            onBlur={handleContentBlur}
            rows={4}
            className="w-full rounded-md border bg-transparent p-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring resize-none"
            disabled={isPending}
          />
          <div className={`text-right text-xs ${isOverLimit ? "text-red-500 font-bold" : "text-muted-foreground"}`}>
            {charCount}/{MAX_CHARS}
          </div>
        </div>

        {imageUrl ? (
          <div className="relative">
            <img src={imageUrl} alt="Attached" className="rounded-md max-h-48 object-cover" />
            <Button
              variant="ghost"
              size="sm"
              onClick={handleRemoveImage}
              className="absolute top-1 right-1 bg-black/50 text-white hover:bg-black/70 h-6 px-2 text-xs"
              disabled={isPending}
            >
              Remove
            </Button>
          </div>
        ) : (
          <label className="inline-flex cursor-pointer items-center gap-2 text-sm text-muted-foreground hover:text-foreground">
            <input
              type="file"
              accept="image/*"
              onChange={handleImageUpload}
              className="hidden"
              disabled={isPending}
            />
            + Add image
          </label>
        )}

        <div className="space-y-2">
          <div className="flex items-center gap-2 text-sm">
            <span className="text-muted-foreground">Schedule:</span>
            {useCustomTime ? (
              <input
                type="datetime-local"
                value={toDatetimeLocal(scheduledAt)}
                onChange={(e) => setScheduledAt(fromDatetimeLocal(e.target.value))}
                className="rounded border bg-transparent px-2 py-1 text-sm"
              />
            ) : (
              <span>
                {new Date(scheduledAt).toLocaleString("ja-JP", { timeZone: "Asia/Tokyo", month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" })}
              </span>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setUseCustomTime(!useCustomTime)}
              className="h-6 px-2 text-xs"
            >
              {useCustomTime ? "Default" : "Change"}
            </Button>
          </div>
        </div>

        {draft.tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {draft.tags.map((tag: string) => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}
          </div>
        )}

        {meta.reply_link && (
          <div className="flex items-center gap-1 text-xs text-muted-foreground">
            <span>🔗</span>
            <a
              href={meta.reply_link}
              target="_blank"
              rel="noopener noreferrer"
              className="truncate hover:underline"
            >
              {meta.reply_link}
            </a>
          </div>
        )}

        <div className="flex gap-2">
          <Button
            onClick={handleApprove}
            disabled={isPending || isOverLimit}
            size="sm"
          >
            Approve
          </Button>
          <Button
            variant="ghost"
            onClick={handleSkip}
            disabled={isPending}
            size="sm"
          >
            Skip
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
