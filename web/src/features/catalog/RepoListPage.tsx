import { useParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { clearAuth } from "@/lib/auth";
import { useRepos } from "./useRepos";
import { RepoTable } from "./RepoTable";

interface RepoListPageProps {
  onLogout: () => void;
}

export function RepoListPage({ onLogout }: RepoListPageProps) {
  const { slug } = useParams<{ slug: string }>();
  const { data: repos, isLoading, error } = useRepos(slug!);

  const handleLogout = () => {
    clearAuth();
    onLogout();
  };

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link to="/dashboard" className="text-xl font-bold tracking-tight hover:text-zinc-300">
              Atlas
            </Link>
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-400">Repositories</span>
            <span className="text-zinc-600">·</span>
            <Link
              to={`/orgs/${slug}/dependencies`}
              className="text-zinc-500 hover:text-zinc-300 text-sm"
            >
              Dependencies
            </Link>
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
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">Repositories</h2>
            {repos && (
              <span className="text-sm text-zinc-500">{repos.length} repositories</span>
            )}
          </div>

          {isLoading && (
            <p className="text-zinc-500 animate-pulse">Loading repositories...</p>
          )}

          {error && (
            <p className="text-red-400">Failed to load repositories.</p>
          )}

          {repos && <RepoTable repos={repos} />}
        </div>
      </main>
    </div>
  );
}
