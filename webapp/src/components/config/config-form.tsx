"use client"

import { useState, useTransition } from "react"
import { Button } from "@/components/ui/button"
import { ConfigField } from "@/components/config/field"
import { saveConfigValues } from "@/app/config/actions"
import type { ConfigRow } from "@/lib/config"

export interface ConfigFormProps {
  rows: ConfigRow[]
}

export function ConfigForm({ rows }: ConfigFormProps) {
  const [pending, startTransition] = useTransition()
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [savedAt, setSavedAt] = useState<number | null>(null)

  return (
    <form
      action={(formData: FormData) => {
        const updates = rows.map((r) => {
          const type = r.metadata.type ?? "string"
          const raw =
            type === "boolean"
              ? formData.get(r.key) !== null
              : (formData.get(r.key) ?? "")
          return { key: r.key, rawValue: raw }
        })
        startTransition(async () => {
          const result = await saveConfigValues(updates)
          if (result.ok) {
            setErrors({})
            setSavedAt(Date.now())
          } else {
            setErrors(result.errors ?? {})
          }
        })
      }}
      className="space-y-6"
    >
      {rows.map((r) => (
        <div key={r.key} className="space-y-1.5">
          <label className="block text-sm font-mono">{r.key}</label>
          {r.metadata.description && (
            <p className="text-xs text-muted-foreground italic whitespace-pre-line">
              {r.metadata.description}
            </p>
          )}
          <ConfigField
            name={r.key}
            type={r.metadata.type ?? "string"}
            defaultValue={r.value}
          />
          {errors[r.key] && (
            <p className="text-xs text-destructive">{errors[r.key]}</p>
          )}
        </div>
      ))}

      <div className="flex items-center gap-3">
        <Button type="submit" disabled={pending}>
          {pending ? "Saving..." : "Save"}
        </Button>
        {savedAt && !pending && Object.keys(errors).length === 0 && (
          <span className="text-sm text-muted-foreground">Saved</span>
        )}
      </div>

      {errors._ && <p className="text-sm text-destructive">{errors._}</p>}
    </form>
  )
}
