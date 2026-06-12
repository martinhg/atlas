import type { User } from "@/lib/auth"
import { clearAuth } from "@/lib/auth"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"

interface Props {
  user: User
  onLogout: () => void
}

export default function DashboardPage({ user, onLogout }: Props) {
  const handleLogout = () => {
    clearAuth()
    onLogout()
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <h1 className="text-xl font-bold tracking-tight">AtlasOS</h1>
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
        <div className="space-y-2">
          <h2 className="text-2xl font-semibold">
            Welcome, {user.name || user.login}
          </h2>
          <p className="text-zinc-400">
            Your engineering intelligence dashboard is coming soon.
          </p>
        </div>
      </main>
    </div>
  )
}
