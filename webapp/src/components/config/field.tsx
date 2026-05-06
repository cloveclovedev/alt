"use client"

import { useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import type { ConfigType } from "@/lib/config"

export interface FieldProps {
  name: string                  // used as the form field name (= the key)
  type: ConfigType | string
  defaultValue: unknown
  cronMinute?: number | null
}

export function ConfigField({ name, type, defaultValue, cronMinute }: FieldProps) {
  switch (type) {
    case "boolean":
      return (
        <label className="inline-flex items-center gap-2">
          <input
            type="checkbox"
            name={name}
            defaultChecked={Boolean(defaultValue)}
            className="h-4 w-4"
          />
          <span className="text-sm text-muted-foreground">enabled</span>
        </label>
      )
    case "number": {
      if (name.endsWith(".cloud.fallback_hour")) {
        return (
          <HourField
            name={name}
            defaultValue={defaultValue}
            cronMinute={cronMinute ?? null}
          />
        )
      }
      return (
        <Input
          type="number"
          name={name}
          defaultValue={
            typeof defaultValue === "number" ? String(defaultValue) : ""
          }
        />
      )
    }
    case "array": {
      if (name.endsWith(".cloud.run_hours")) {
        return (
          <RunHoursField
            name={name}
            defaultValue={defaultValue}
            cronMinute={cronMinute ?? null}
          />
        )
      }
      return <ArrayEditor name={name} defaultValue={defaultValue} />
    }
    case "object": {
      const json = JSON.stringify(defaultValue ?? {}, null, 2)
      return (
        <textarea
          name={name}
          defaultValue={json}
          rows={Math.min(15, json.split("\n").length + 1)}
          className="w-full rounded-md border bg-background p-2 font-mono text-sm"
        />
      )
    }
    case "string":
    default: {
      const val = defaultValue == null ? "" : String(defaultValue)
      if (val.includes("\n")) {
        return (
          <textarea
            name={name}
            defaultValue={val}
            rows={Math.min(10, val.split("\n").length + 1)}
            className="w-full rounded-md border bg-background p-2 text-sm"
          />
        )
      }
      return <Input type="text" name={name} defaultValue={val} />
    }
  }
}

interface ArrayEditorProps {
  name: string
  defaultValue: unknown
}

function ArrayEditor({ name, defaultValue }: ArrayEditorProps) {
  const initial = Array.isArray(defaultValue) ? defaultValue.map(String) : []
  const [items, setItems] = useState<string[]>(initial)

  return (
    <div className="space-y-2">
      {items.map((item, idx) => (
        <div key={idx} className="flex items-center gap-2">
          <Input
            value={item}
            onChange={(e) => {
              const next = [...items]
              next[idx] = e.target.value
              setItems(next)
            }}
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => setItems(items.filter((_, i) => i !== idx))}
            aria-label="Remove item"
          >
            −
          </Button>
        </div>
      ))}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => setItems([...items, ""])}
      >
        + Add item
      </Button>
      {/*
        Hidden field carrying the JSON form value. The Server Action's
        cast for type=array parses this back via JSON.parse. readOnly
        because it is controlled-without-handler by design.
      */}
      <input type="hidden" name={name} value={JSON.stringify(items)} readOnly />
    </div>
  )
}

interface HourFieldProps {
  name: string
  defaultValue: unknown
  cronMinute: number | null
}

function HourField({ name, defaultValue, cronMinute }: HourFieldProps) {
  const initial = typeof defaultValue === "number" ? defaultValue : null
  const [hour, setHour] = useState<number | null>(initial)
  return (
    <div className="space-y-1">
      <Input
        type="number"
        min={0}
        max={23}
        name={name}
        value={hour == null ? "" : String(hour)}
        onChange={(e) => {
          const v = e.target.value
          setHour(v === "" ? null : Number(v))
        }}
      />
      <FireTimeCaption hours={hour == null ? [] : [hour]} cronMinute={cronMinute} />
    </div>
  )
}

interface RunHoursFieldProps {
  name: string
  defaultValue: unknown
  cronMinute: number | null
}

function RunHoursField({ name, defaultValue, cronMinute }: RunHoursFieldProps) {
  const initial = Array.isArray(defaultValue)
    ? defaultValue.filter((x): x is number => typeof x === "number")
    : []
  const [hours, setHours] = useState<number[]>(initial)
  return (
    <div className="space-y-2">
      {hours.map((h, idx) => (
        <div key={idx} className="flex items-center gap-2">
          <Input
            type="number"
            min={0}
            max={23}
            value={String(h)}
            onChange={(e) => {
              const v = Number(e.target.value)
              const next = [...hours]
              next[idx] = isNaN(v) ? 0 : v
              setHours(next)
            }}
          />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => setHours(hours.filter((_, i) => i !== idx))}
            aria-label="Remove hour"
          >
            −
          </Button>
        </div>
      ))}
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => setHours([...hours, 0])}
      >
        + Add hour
      </Button>
      <input
        type="hidden"
        name={name}
        value={JSON.stringify(hours)}
        readOnly
      />
      <FireTimeCaption hours={hours} cronMinute={cronMinute} />
    </div>
  )
}

interface FireTimeCaptionProps {
  hours: number[]
  cronMinute: number | null
}

function FireTimeCaption({ hours, cronMinute }: FireTimeCaptionProps) {
  if (cronMinute == null) {
    return (
      <p className="text-xs text-muted-foreground">
        Effective fire time: minute unset — set cloud_scheduler.cron_minute first.
      </p>
    )
  }
  if (hours.length === 0) {
    return (
      <p className="text-xs text-muted-foreground">
        Effective fire time: (set hour above)
      </p>
    )
  }
  const sorted = [...new Set(hours)].sort((a, b) => a - b)
  const mm = String(cronMinute).padStart(2, "0")
  const list = sorted
    .map((h) => `${String(h).padStart(2, "0")}:${mm}`)
    .join(", ")
  return (
    <p className="text-xs text-muted-foreground">
      Effective fire time(s) JST: {list}
    </p>
  )
}
