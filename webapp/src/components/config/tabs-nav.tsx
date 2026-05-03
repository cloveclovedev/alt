import Link from "next/link"

export interface TabsNavProps {
  tabs: string[]
  active: string
}

export function TabsNav({ tabs, active }: TabsNavProps) {
  return (
    <nav className="border-b mb-6">
      <ul className="flex gap-1">
        {tabs.map((t) => {
          const isActive = t === active
          return (
            <li key={t}>
              <Link
                href={`/config?tab=${encodeURIComponent(t)}`}
                className={
                  "inline-block px-3 py-2 text-sm border-b-2 -mb-[1px] " +
                  (isActive
                    ? "border-foreground text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground")
                }
              >
                {t}
              </Link>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
