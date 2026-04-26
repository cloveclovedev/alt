import { getActiveGoals, getRecentMemos, getUpcomingDeadlines } from "@/lib/queries"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { BodySummaryCard } from "@/components/body/body-summary-card"

export default async function DashboardPage() {
  const [goals, memos, deadlines] = await Promise.all([
    getActiveGoals(),
    getRecentMemos(7),
    getUpcomingDeadlines(7),
  ])

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">Dashboard</h1>

      {deadlines.length > 0 && (
        <Card className="mb-6 border-amber-500">
          <CardHeader>
            <CardTitle className="text-amber-500">Deadline Alerts</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2">
              {deadlines.map((entry) => (
                <li key={entry.id} className="flex items-center justify-between">
                  <span className="font-medium">{entry.title}</span>
                  <Badge variant="destructive">
                    {(entry.metadata as { target_date?: string }).target_date}
                  </Badge>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Active Goals</CardTitle>
          </CardHeader>
          <CardContent>
            {goals.length === 0 ? (
              <p className="text-muted-foreground">No active goals</p>
            ) : (
              <ul className="space-y-3">
                {goals.map((goal) => (
                  <li key={goal.id}>
                    <p className="font-medium">{goal.title}</p>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Recent Memos</CardTitle>
          </CardHeader>
          <CardContent>
            {memos.length === 0 ? (
              <p className="text-muted-foreground">No recent memos</p>
            ) : (
              <ul className="space-y-3">
                {memos.map((memo) => (
                  <li key={memo.id}>
                    <p className="font-medium">{memo.title}</p>
                    {memo.content && (
                      <p className="mt-0.5 text-sm text-muted-foreground line-clamp-2">
                        {memo.content}
                      </p>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="mt-6">
        <BodySummaryCard />
      </div>
    </div>
  )
}
