"use client"

import { useRouter, useSearchParams } from "next/navigation"
import { Button } from "@/components/ui/button"
import type { Period } from "@/lib/types"

const PERIODS: { value: Period; label: string }[] = [
  { value: "30d", label: "30D" },
  { value: "90d", label: "90D" },
  { value: "6m", label: "6M" },
  { value: "1y", label: "1Y" },
  { value: "all", label: "All" },
]

export function PeriodSelector() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const current = (searchParams.get("period") ?? "90d") as Period

  function handleSelect(period: Period) {
    const params = new URLSearchParams(searchParams.toString())
    params.set("period", period)
    router.push(`?${params.toString()}`)
  }

  return (
    <div className="flex gap-1">
      {PERIODS.map(({ value, label }) => (
        <Button
          key={value}
          variant={current === value ? "default" : "ghost"}
          size="sm"
          onClick={() => handleSelect(value)}
        >
          {label}
        </Button>
      ))}
    </div>
  )
}
