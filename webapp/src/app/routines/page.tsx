import { getLatestRoutineEntries } from "@/lib/queries"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"

export default async function RoutinesPage() {
  const events = await getLatestRoutineEntries()
  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="text-3xl font-bold mb-6">Routines</h1>
      {events.length === 0 ? (
        <p className="text-muted-foreground">No routine events recorded.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Routine</TableHead>
              <TableHead className="w-32">Category</TableHead>
              <TableHead className="w-32">Kind</TableHead>
              <TableHead className="w-40">Last Completed</TableHead>
              <TableHead>Note</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.map((event) => (
              <TableRow key={event.id}>
                <TableCell className="font-medium">{event.title}</TableCell>
                <TableCell>
                  <Badge variant="outline">{String(event.metadata.category ?? "")}</Badge>
                </TableCell>
                <TableCell>
                  <Badge variant="secondary">{event.status ?? ""}</Badge>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {event.metadata.completed_at
                    ? new Date(String(event.metadata.completed_at)).toLocaleDateString()
                    : "—"}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {event.content || "—"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
