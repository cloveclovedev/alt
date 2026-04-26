"use client"

import { useRouter, useSearchParams } from "next/navigation"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"

export function EntryFilters() {
  const router = useRouter()
  const searchParams = useSearchParams()

  function updateFilter(key: string, value: string) {
    const params = new URLSearchParams(searchParams.toString())
    if (value && value !== "all") {
      params.set(key, value)
    } else {
      params.delete(key)
    }
    router.push(`/entries?${params.toString()}`)
  }

  return (
    <div className="flex flex-wrap gap-3">
      <Select
        value={searchParams.get("type") || "all"}
        onValueChange={(v) => updateFilter("type", v as string)}
      >
        <SelectTrigger className="w-40">
          <SelectValue placeholder="Type" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All Types</SelectItem>
          <SelectItem value="memo">Memo</SelectItem>
          <SelectItem value="goal">Goal</SelectItem>
          <SelectItem value="knowledge">Knowledge</SelectItem>
          <SelectItem value="tech_interest">Tech Interest</SelectItem>
          <SelectItem value="business">Business</SelectItem>
        </SelectContent>
      </Select>

      <Select
        value={searchParams.get("status") || "all"}
        onValueChange={(v) => updateFilter("status", v as string)}
      >
        <SelectTrigger className="w-40">
          <SelectValue placeholder="Status" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All Statuses</SelectItem>
          <SelectItem value="active">Active</SelectItem>
          <SelectItem value="achieved">Achieved</SelectItem>
          <SelectItem value="dropped">Dropped</SelectItem>
        </SelectContent>
      </Select>

      <form
        className="flex-1 min-w-[200px]"
        onSubmit={(e) => {
          e.preventDefault()
          const formData = new FormData(e.currentTarget)
          updateFilter("q", formData.get("q") as string)
        }}
      >
        <Input
          name="q"
          placeholder="Search entries..."
          defaultValue={searchParams.get("q") || ""}
        />
      </form>
    </div>
  )
}
