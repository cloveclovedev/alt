"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { METRIC_CONFIG } from "@/lib/body-utils"
import type { BodyGoal, BodyGoalStatus } from "@/lib/types"
import { Badge } from "@/components/ui/badge"

interface CompositionGoalCardProps {
  weightGoal?: BodyGoal | null
  skmGoal?: BodyGoal | null
  derivedBfPct?: number | null
  derivedFfmi?: number | null
  pastGoals: BodyGoal[]
}

export function CompositionGoalCard({
  weightGoal,
  skmGoal,
  derivedBfPct,
  derivedFfmi,
  pastGoals,
}: CompositionGoalCardProps) {
  const router = useRouter()
  const [editing, setEditing] = useState(false)
  const [creating, setCreating] = useState(false)
  const [showHistory, setShowHistory] = useState(false)

  const hasActive = !!(weightGoal || skmGoal)
  const targetDate = weightGoal?.target_date ?? skmGoal?.target_date ?? ""

  async function handleSave(formData: FormData) {
    const target_date = formData.get("target_date") as string
    const weight = Number(formData.get("weight"))
    const skm = Number(formData.get("skm"))

    if (hasActive) {
      // Update existing goals
      const updates = []
      if (weightGoal) {
        updates.push(
          fetch(`/api/body/goals/${weightGoal.id}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ target_value: weight, target_date }),
          })
        )
      } else {
        updates.push(
          fetch("/api/body/goals", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ metric: "weight_kg", target_value: weight, target_date }),
          })
        )
      }
      if (skmGoal) {
        updates.push(
          fetch(`/api/body/goals/${skmGoal.id}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ target_value: skm, target_date }),
          })
        )
      } else {
        updates.push(
          fetch("/api/body/goals", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ metric: "skeletal_muscle_mass_kg", target_value: skm, target_date }),
          })
        )
      }
      await Promise.all(updates)
    } else {
      // Create both goals
      await Promise.all([
        fetch("/api/body/goals", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ metric: "weight_kg", target_value: weight, target_date }),
        }),
        fetch("/api/body/goals", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ metric: "skeletal_muscle_mass_kg", target_value: skm, target_date }),
        }),
      ])
    }
    setEditing(false)
    setCreating(false)
    router.refresh()
  }

  async function handleStatusChange(status: BodyGoalStatus) {
    const updates = []
    if (weightGoal) {
      updates.push(
        fetch(`/api/body/goals/${weightGoal.id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ status }),
        })
      )
    }
    if (skmGoal) {
      updates.push(
        fetch(`/api/body/goals/${skmGoal.id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ status }),
        })
      )
    }
    await Promise.all(updates)
    router.refresh()
  }

  const showForm = editing || creating

  return (
    <div className="rounded-lg border p-4 col-span-full">
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold text-base">Body Composition Goal</h3>
        {hasActive && !showForm && (
          <div className="flex gap-2">
            <Button size="sm" variant="ghost" onClick={() => setEditing(true)}>Edit</Button>
            <Button size="sm" variant="ghost" onClick={() => handleStatusChange("achieved")}>Achieved</Button>
            <Button size="sm" variant="ghost" onClick={() => handleStatusChange("expired")}>Expired</Button>
          </div>
        )}
      </div>

      {hasActive && !showForm && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div>
            <div className="text-xs text-muted-foreground">Deadline</div>
            <div className="font-medium">{targetDate}</div>
          </div>
          <div>
            <div className="text-xs" style={{ color: METRIC_CONFIG.weight_kg.color }}>Weight</div>
            <div className="font-medium">
              {weightGoal ? `${weightGoal.target_value} kg` : "—"}
            </div>
          </div>
          <div>
            <div className="text-xs" style={{ color: METRIC_CONFIG.skeletal_muscle_mass_kg.color }}>Skeletal Muscle</div>
            <div className="font-medium">
              {skmGoal ? `${skmGoal.target_value} kg` : "—"}
            </div>
          </div>
          {derivedBfPct != null && derivedFfmi != null && (
            <>
              <div className="col-span-2 md:col-span-4 border-t pt-2 mt-1">
                <div className="text-xs text-muted-foreground mb-1">Estimated from targets</div>
                <div className="flex gap-6">
                  <span>
                    <span className="text-xs" style={{ color: METRIC_CONFIG.body_fat_percent.color }}>Body Fat % </span>
                    <strong>{derivedBfPct}%</strong>
                  </span>
                  <span>
                    <span className="text-xs" style={{ color: METRIC_CONFIG.ffmi.color }}>FFMI </span>
                    <strong>{derivedFfmi}</strong>
                  </span>
                </div>
              </div>
            </>
          )}
        </div>
      )}

      {showForm && (
        <form action={handleSave} className="space-y-3">
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="text-xs text-muted-foreground">Deadline</label>
              <input
                name="target_date"
                type="date"
                defaultValue={targetDate}
                className="w-full rounded-md border px-2 py-1 text-sm bg-background"
                required
              />
            </div>
            <div>
              <label className="text-xs" style={{ color: METRIC_CONFIG.weight_kg.color }}>Weight (kg)</label>
              <input
                name="weight"
                type="number"
                step="0.1"
                defaultValue={weightGoal?.target_value ?? ""}
                className="w-full rounded-md border px-2 py-1 text-sm bg-background"
                required
              />
            </div>
            <div>
              <label className="text-xs" style={{ color: METRIC_CONFIG.skeletal_muscle_mass_kg.color }}>Skeletal Muscle (kg)</label>
              <input
                name="skm"
                type="number"
                step="0.1"
                defaultValue={skmGoal?.target_value ?? ""}
                className="w-full rounded-md border px-2 py-1 text-sm bg-background"
                required
              />
            </div>
          </div>
          <div className="flex gap-2">
            <Button size="sm" type="submit">{hasActive ? "Save" : "Create"}</Button>
            <Button size="sm" variant="ghost" type="button" onClick={() => { setEditing(false); setCreating(false) }}>Cancel</Button>
          </div>
        </form>
      )}

      {!hasActive && !showForm && (
        <Button size="sm" variant="ghost" onClick={() => setCreating(true)}>
          Set goal
        </Button>
      )}

      {pastGoals.length > 0 && (
        <div className="mt-3">
          <button
            className="text-xs text-muted-foreground hover:text-foreground"
            onClick={() => setShowHistory(!showHistory)}
          >
            {showHistory ? "Hide" : "Show"} history ({pastGoals.length})
          </button>
          {showHistory && (
            <ul className="mt-2 space-y-1">
              {pastGoals.map((g) => {
                const config = METRIC_CONFIG[g.metric as keyof typeof METRIC_CONFIG]
                return (
                  <li key={g.id} className="text-xs text-muted-foreground flex items-center gap-2">
                    <Badge variant="secondary">{g.status}</Badge>
                    {config?.label}: {g.target_value}{config?.unit ? ` ${config.unit}` : ""} by {g.target_date}
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
