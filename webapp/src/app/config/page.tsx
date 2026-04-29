import { listConfigsWithMeta } from "@/lib/config"
import type { ConfigRow } from "@/lib/config"
import { ConfigForm } from "@/components/config/config-form"
import { TabsNav } from "@/components/config/tabs-nav"

const PHASE_1_SKILLS = ["daily-plan"] as const

interface SearchParams {
  tab?: string
}

export default async function ConfigPage(props: {
  searchParams: Promise<SearchParams>
}) {
  const sp = await props.searchParams
  const all = await listConfigsWithMeta()
  const tabs = computeTabs(all)
  const active = sp.tab && tabs.includes(sp.tab) ? sp.tab : tabs[0]
  const rows = filterByTab(all, active)

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <h1 className="text-3xl font-bold mb-6">Config</h1>

      <TabsNav tabs={tabs} active={active} />

      {rows.length === 0 ? (
        <p className="text-muted-foreground">No params for this tab.</p>
      ) : (
        <ConfigForm rows={rows} />
      )}
    </div>
  )
}

function computeTabs(rows: ConfigRow[]): string[] {
  const owned = new Set<string>()
  let hasUnowned = false
  for (const r of rows) {
    const consumers = r.metadata.consumed_by ?? []
    if (consumers.length === 0) {
      if (!r.key.startsWith("_")) hasUnowned = true
      continue
    }
    for (const c of consumers) owned.add(c)
  }
  // Phase 1 only surfaces daily-plan related skills + Custom; restrict here so
  // we do not accidentally show weekly-plan, x-draft, etc. that have not been
  // designed yet.
  const phase1 = PHASE_1_SKILLS.filter((s) =>
    Array.from(owned).some((o) => o === s || o.startsWith(s + "-") || o === s + "-cloud")
  )
  return [...phase1, ...(hasUnowned ? ["Custom"] : [])]
}

function filterByTab(rows: ConfigRow[], tab: string): ConfigRow[] {
  if (tab === "Custom") {
    return rows.filter(
      (r) => !r.key.startsWith("_") && (r.metadata.consumed_by ?? []).length === 0
    )
  }
  return rows.filter((r) => {
    const consumers = r.metadata.consumed_by ?? []
    return consumers.some(
      (c) => c === tab || c.startsWith(tab + "-") || c === tab + "-cloud"
    )
  })
}
