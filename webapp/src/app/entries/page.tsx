import { Suspense } from "react"
import { listEntries } from "@/lib/queries"
import { EntryFilters } from "@/components/entry-filters"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"

export default async function EntriesPage(props: {
  searchParams: Promise<{
    type?: string
    status?: string
    q?: string
  }>
}) {
  const searchParams = await props.searchParams
  const entries = await listEntries({
    type: searchParams.type || null,
    status: searchParams.status || null,
    search: searchParams.q || null,
  })

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="text-3xl font-bold mb-6">Entries</h1>

      <Suspense fallback={null}>
        <EntryFilters />
      </Suspense>

      <div className="mt-6">
        {entries.length === 0 ? (
          <p className="text-muted-foreground">No entries found.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Title</TableHead>
                <TableHead className="w-28">Type</TableHead>
                <TableHead className="w-28">Status</TableHead>
                <TableHead className="w-32">Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {entries.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell>
                    <p className="font-medium">{entry.title}</p>
                    {entry.content && (
                      <p className="text-sm text-muted-foreground line-clamp-1">
                        {entry.content}
                      </p>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{entry.type}</Badge>
                  </TableCell>
                  <TableCell>
                    {entry.status && <Badge variant="secondary">{entry.status}</Badge>}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {new Date(entry.created_at).toLocaleDateString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  )
}
