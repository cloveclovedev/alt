"use client"

import { useState, useEffect } from "react"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import type { BodyMeasurement } from "@/lib/types"

const PAGE_SIZE = 10

const DISPLAY_COLUMNS: { key: keyof BodyMeasurement; label: string }[] = [
  { key: "measured_at", label: "Date" },
  { key: "weight_kg", label: "Weight" },
  { key: "body_fat_percent", label: "BF%" },
  { key: "muscle_mass_kg", label: "Muscle" },
  { key: "ffmi", label: "FFMI" },
  { key: "bmi", label: "BMI" },
  { key: "inbody_score", label: "Score" },
]

interface HistoryData {
  rows: BodyMeasurement[]
  total: number
}

export function MeasurementHistory() {
  const [data, setData] = useState<HistoryData | null>(null)
  const [page, setPage] = useState(0)

  useEffect(() => {
    async function fetchData() {
      const offset = page * PAGE_SIZE
      const res = await fetch(
        `/api/body/measurements?period=all&limit=${PAGE_SIZE}&offset=${offset}`
      )
      const json = await res.json()
      setData(json)
    }
    fetchData()
  }, [page])

  if (!data) return <p className="text-muted-foreground">Loading...</p>

  const totalPages = Math.ceil(data.total / PAGE_SIZE)

  return (
    <div>
      <h2 className="text-lg font-semibold mb-3">Measurement History</h2>
      <Table>
        <TableHeader>
          <TableRow>
            {DISPLAY_COLUMNS.map((col) => (
              <TableHead key={col.key}>{col.label}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.rows.map((row) => (
            <TableRow key={row.id}>
              {DISPLAY_COLUMNS.map((col) => (
                <TableCell key={col.key}>
                  {col.key === "measured_at"
                    ? new Date(row.measured_at).toLocaleDateString()
                    : (row[col.key] as number | null) ?? "—"}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-3">
          <Button
            size="sm"
            variant="ghost"
            disabled={page === 0}
            onClick={() => setPage(page - 1)}
          >
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {page + 1} of {totalPages}
          </span>
          <Button
            size="sm"
            variant="ghost"
            disabled={page >= totalPages - 1}
            onClick={() => setPage(page + 1)}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  )
}
