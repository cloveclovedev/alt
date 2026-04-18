import { getXDrafts, getOccupiedSlots } from "@/lib/queries"
import { getNextAvailableSlot } from "@/lib/scheduling"
import { DraftCard } from "@/components/posts/draft-card"
import { ApprovedCard } from "@/components/posts/approved-card"
import { HistoryCard } from "@/components/posts/history-card"
import type { Entry } from "@/lib/types"

const DEFAULT_TIMES = ["12:00", "19:00"]

export default async function PostsPage() {
  const [drafts, occupiedSlots] = await Promise.all([
    getXDrafts(),
    getOccupiedSlots(),
  ])

  const now = new Date()
  const nextSlot = getNextAvailableSlot(now, occupiedSlots, DEFAULT_TIMES)

  const draftEntries = drafts.filter((d: Entry) => d.status === "draft")
  const approvedEntries = drafts.filter((d: Entry) => d.status === "approved")
  const historyEntries = drafts.filter((d: Entry) => d.status === "posted" || d.status === "skipped")

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">Posts</h1>

      {draftEntries.length > 0 && (
        <section className="mb-8">
          <h2 className="text-lg font-semibold mb-4">Drafts</h2>
          <div className="space-y-4">
            {draftEntries.map((draft: Entry) => (
              <DraftCard
                key={draft.id}
                draft={draft}
                nextSlot={nextSlot}
                defaultTimes={DEFAULT_TIMES}
                occupiedSlots={occupiedSlots}
              />
            ))}
          </div>
        </section>
      )}

      {approvedEntries.length > 0 && (
        <section className="mb-8">
          <h2 className="text-lg font-semibold mb-4">Scheduled</h2>
          <div className="space-y-4">
            {approvedEntries.map((draft: Entry) => (
              <ApprovedCard key={draft.id} draft={draft} />
            ))}
          </div>
        </section>
      )}

      {historyEntries.length > 0 && (
        <section>
          <details>
            <summary className="text-lg font-semibold mb-4 cursor-pointer">History</summary>
            <div className="space-y-4 mt-4">
              {historyEntries.map((draft: Entry) => (
                <HistoryCard key={draft.id} draft={draft} />
              ))}
            </div>
          </details>
        </section>
      )}

      {drafts.length === 0 && (
        <p className="text-muted-foreground">No drafts yet. Drafts are generated automatically at 10:00 and 18:00.</p>
      )}
    </div>
  )
}
