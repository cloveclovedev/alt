import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import type { Entry } from "@/lib/types"

interface HistoryCardProps {
  draft: Entry
}

export function HistoryCard({ draft }: HistoryCardProps) {
  const metadata = draft.metadata as { posted_at?: string; tweet_id?: string; image_url?: string }
  const isPosted = draft.status === "posted"

  return (
    <Card className="opacity-60">
      <CardContent className="space-y-2 pt-6">
        <div className="flex items-center justify-between">
          <Badge variant={isPosted ? "secondary" : "outline"}>
            {isPosted ? "Posted" : "Skipped"}
          </Badge>
          <span className="text-xs text-muted-foreground">
            {metadata.posted_at
              ? new Date(metadata.posted_at).toLocaleString("ja-JP", { timeZone: "Asia/Tokyo" })
              : new Date(draft.updated_at).toLocaleString("ja-JP", { timeZone: "Asia/Tokyo" })}
          </span>
        </div>

        <p className="text-sm whitespace-pre-wrap">{draft.content}</p>

        {metadata.image_url && (
          <img src={metadata.image_url} alt="Attached" className="rounded-md max-h-32 object-cover" />
        )}

        {isPosted && metadata.tweet_id && (
          <a
            href={`https://x.com/i/status/${metadata.tweet_id}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-blue-500 hover:underline"
          >
            View on X
          </a>
        )}
      </CardContent>
    </Card>
  )
}
