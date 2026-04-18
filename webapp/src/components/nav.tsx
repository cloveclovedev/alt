import Link from "next/link"
import { auth, signOut } from "@/lib/auth"
import { Button } from "@/components/ui/button"

export async function Nav() {
  const session = await auth()

  if (!session?.user) return null

  return (
    <nav className="border-b">
      <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
        <div className="flex items-center gap-6">
          <Link href="/" className="text-lg font-bold">
            alt
          </Link>
          <Link href="/entries" className="text-sm text-muted-foreground hover:text-foreground">
            Entries
          </Link>
          <Link href="/routines" className="text-sm text-muted-foreground hover:text-foreground">
            Routines
          </Link>
          <Link href="/body" className="text-sm text-muted-foreground hover:text-foreground">
            Body
          </Link>
          <Link href="/posts" className="text-sm text-muted-foreground hover:text-foreground">
            Posts
          </Link>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">{session.user.name}</span>
          <form
            action={async () => {
              "use server"
              await signOut({ redirectTo: "/login" })
            }}
          >
            <Button variant="ghost" size="sm" type="submit">
              Sign out
            </Button>
          </form>
        </div>
      </div>
    </nav>
  )
}
