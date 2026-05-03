"use client"

import { useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import type { ConfigType } from "@/lib/config"

export interface FieldProps {
  name: string                  // used as the form field name (= the key)
  type: ConfigType | string
  defaultValue: unknown
}

export function ConfigField({ name, type, defaultValue }: FieldProps) {
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
    case "number":
      return (
        <Input
          type="number"
          name={name}
          defaultValue={
            typeof defaultValue === "number" ? String(defaultValue) : ""
          }
        />
      )
    case "array":
      return <ArrayEditor name={name} defaultValue={defaultValue} />
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
