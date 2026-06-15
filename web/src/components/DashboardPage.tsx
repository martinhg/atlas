import { Link } from "react-router-dom"
import type { User } from "@/lib/auth"
import { clearAuth } from "@/lib/auth"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { useOrgs } from "@/hooks/useOrgs"

const GITHUB_APP_SLUG = import.meta.env.VITE_GITHUB_APP_SLUG || "atlas-dev"

interface Props {
  user: User
  onLogout: () => void
}

export default function DashboardPage({ user, onLogout }: Props) {
  const { data: orgs, isLoading: orgsLoading } = useOrgs()

  const handleLogout = () => {
    clearAuth()
    onLogout()
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <h1 className="text-xl font-bold tracking-tight">Atlas</h1>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-3">
              <Avatar>
                <AvatarImage src={user.avatar_url} alt={user.login} />
                <AvatarFallback>{user.login.slice(0, 2).toUpperCase()}</AvatarFallback>
              </Avatar>
              <span className="text-sm text-zinc-300">{user.login}</span>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleLogout}
              className="text-zinc-500 hover:text-zinc-300"
            >
              Sign out
            </Button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-12">
        <div className="space-y-6">
          <div className="space-y-2">
            <h2 className="text-2xl font-semibold">
              Welcome, {user.name || user.login}
            </h2>
            <p className="text-zinc-400">
              Your engineering intelligence dashboard.
            </p>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-medium">Organizations</h3>
              <Button asChild size="sm" variant="outline" className="border-zinc-700 text-zinc-300 hover:bg-zinc-800">
                <a href={`https://github.com/apps/${GITHUB_APP_SLUG}/installations/new`}>
                  Connect GitHub
                </a>
              </Button>
            </div>

            {orgsLoading && (
              <p className="text-zinc-500 text-sm animate-pulse">Loading organizations...</p>
            )}

            {orgs && orgs.length === 0 && (
              <p className="text-zinc-500 text-sm">
                No organizations connected yet. Click "Connect GitHub" to get started.
              </p>
            )}

            {orgs && orgs.length > 0 && (
              <div className="grid gap-3">
                {orgs.map((org) => (
                  <div
                    key={org.id}
                    className="p-4 rounded-lg border border-zinc-800 space-y-3"
                  >
                    <div>
                      <p className="font-medium">{org.name}</p>
                      <p className="text-sm text-zinc-500">{org.slug}</p>
                    </div>
                    <div className="flex items-center gap-3">
                      <Link
                        to={`/orgs/${org.slug}/repos`}
                        className="text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
                      >
                        Repositories →
                      </Link>
                      <Link
                        to={`/orgs/${org.slug}/dependencies`}
                        className="text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
                      >
                        Dependencies →
                      </Link>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
