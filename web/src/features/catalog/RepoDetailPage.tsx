import { useParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { clearAuth } from "@/lib/auth";
import { useRepoDetail } from "./useRepoDetail";
import { useRepoDeps } from "./useRepoDeps";
import { useOwnershipDetail } from "@/features/ownership/useOwnershipDetail";

interface Props {
  onLogout: () => void;
}

export function RepoDetailPage({ onLogout }: Props) {
  const { slug, name } = useParams<{ slug: string; name: string }>();

  const repo = useRepoDetail(slug!, name!);
  const deps = useRepoDeps(slug!, name!);
  const owners = useOwnershipDetail(slug!, name!);

  const handleLogout = () => {
    clearAuth();
    onLogout();
  };

  const isLoading = repo.isPending || deps.isPending || owners.isPending;

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link to="/dashboard" className="text-xl font-bold tracking-tight hover:text-zinc-300">
              Atlas
            </Link>
            <span className="text-zinc-600">/</span>
            <Link
              to={`/orgs/${slug}/repos`}
              className="text-zinc-400 hover:text-zinc-200"
            >
              Repositories
            </Link>
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-300">{name}</span>
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
        <div className="space-y-8">
          {isLoading && (
            <p className="text-zinc-500 animate-pulse">Loading repository details...</p>
          )}

          {repo.isError && (
            <p className="text-red-400">Failed to load repository.</p>
          )}

          {repo.data && (
            <div className="space-y-2">
              <h2 className="text-2xl font-semibold">{repo.data.full_name}</h2>
              {repo.data.description && (
                <p className="text-zinc-400">{repo.data.description}</p>
              )}
              <div className="flex items-center gap-4 text-sm text-zinc-500">
                {repo.data.language && <span>{repo.data.language}</span>}
                <span>{repo.data.default_branch}</span>
                <span>{repo.data.stars} stars</span>
                {repo.data.private && (
                  <span className="text-amber-500/80">Private</span>
                )}
                {repo.data.fork && (
                  <span className="text-zinc-600">Fork</span>
                )}
              </div>
            </div>
          )}

          {deps.data && (
            <section className="space-y-3">
              <h3 className="text-lg font-medium text-zinc-200">
                Dependencies
                <span className="ml-2 text-sm text-zinc-500">
                  {deps.data.dependencies.length}
                </span>
              </h3>
              {deps.data.dependencies.length === 0 ? (
                <p className="text-zinc-500 text-sm">No dependencies found.</p>
              ) : (
                <div className="border border-zinc-800 rounded-lg overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-zinc-800 bg-zinc-900/50">
                        <th className="text-left px-4 py-3 font-medium text-zinc-400">Name</th>
                        <th className="text-left px-4 py-3 font-medium text-zinc-400">Ecosystem</th>
                        <th className="text-left px-4 py-3 font-medium text-zinc-400">Version</th>
                        <th className="text-left px-4 py-3 font-medium text-zinc-400">Type</th>
                        <th className="text-left px-4 py-3 font-medium text-zinc-400">Source</th>
                      </tr>
                    </thead>
                    <tbody>
                      {deps.data.dependencies.map((dep, i) => (
                        <tr
                          key={`${dep.ecosystem}-${dep.name}-${dep.source_file}-${i}`}
                          className="border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30"
                        >
                          <td className="px-4 py-3 font-medium text-zinc-100">
                            <Link
                              to={`/orgs/${slug}/dependencies/${dep.ecosystem}/${dep.name}`}
                              className="hover:text-zinc-300 hover:underline"
                            >
                              {dep.name}
                            </Link>
                          </td>
                          <td className="px-4 py-3 text-zinc-400">{dep.ecosystem}</td>
                          <td className="px-4 py-3 text-zinc-400">{dep.version || "—"}</td>
                          <td className="px-4 py-3 text-zinc-400">{dep.dep_type}</td>
                          <td className="px-4 py-3 text-zinc-500 text-xs">{dep.source_file}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          )}

          {owners.data && (
            <section className="space-y-3">
              <h3 className="text-lg font-medium text-zinc-200">
                Ownership
                <span className="ml-2 text-sm text-zinc-500">
                  {owners.data.rules.length} rules
                </span>
              </h3>
              {owners.data.rules.length === 0 ? (
                <p className="text-zinc-500 text-sm">No CODEOWNERS rules found.</p>
              ) : (
                <div className="border border-zinc-800 rounded-lg overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-zinc-800 bg-zinc-900/50">
                        <th className="text-left px-4 py-3 font-medium text-zinc-400">Pattern</th>
                        <th className="text-left px-4 py-3 font-medium text-zinc-400">Owner</th>
                        <th className="text-left px-4 py-3 font-medium text-zinc-400">Type</th>
                      </tr>
                    </thead>
                    <tbody>
                      {owners.data.rules.map((rule, i) => (
                        <tr
                          key={`${rule.pattern}-${rule.owner}-${i}`}
                          className="border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30"
                        >
                          <td className="px-4 py-3 font-mono text-zinc-300">{rule.pattern}</td>
                          <td className="px-4 py-3 text-zinc-100">{rule.owner}</td>
                          <td className="px-4 py-3 text-zinc-400">{rule.owner_type}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          )}
        </div>
      </main>
    </div>
  );
}
