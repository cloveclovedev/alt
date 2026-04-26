"use client"

import { useTransition } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { cancelApproval } from "@/app/posts/actions"
import type { Entry } from "@/lib/types"

interface ApprovedCardProps {
  draft: Entry
}

export function ApprovedCard({ draft }: ApprovedCardProps) {
  const [isPending, startTransition] = useTransition()
  const metadata = draft.metadata as { scheduled_at?: string; image_url?: string }

  function handleCancel() {
    startTransition(() => cancelApproval(draft.id))
  }

  return (
    <Card className="border-blue-500/30">
      <CardContent className="space-y-3 pt-6">
        <div className="flex items-center justify-between">
          <Badge className="bg-blue-500/10 text-blue-500 border-blue-500/30">Scheduled</Badge>
          {metadata.scheduled_at && (
            <span className="text-xs font-medium">
              {new Date(metadata.scheduled_at).toLocaleString("ja-JP", {
                timeZone: "Asia/Tokyo",
                month: "numeric",
                day: "numeric",
                hour: "2-digit",
                minute: "2-digit",
              })}
            </span>
          )}
        </div>

        <p className="text-sm whitespace-pre-wrap">{draft.content}</p>

        {metadata.image_url && (
          <img src={metadata.image_url} alt="Attached" className="rounded-md max-h-48 object-cover" />
        )}

        <Button variant="ghost" size="sm" onClick={handleCancel} disabled={isPending}>
          Cancel
        </Button>
      </CardContent>
    </Card>
  )
}
