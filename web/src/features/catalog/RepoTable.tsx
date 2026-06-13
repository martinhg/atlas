import type { Repository } from "@/lib/api";

interface RepoTableProps {
  repos: Repository[];
}

export function RepoTable({ repos }: RepoTableProps) {
  if (repos.length === 0) {
    return (
      <p className="text-zinc-500 text-sm py-8 text-center">
        No repositories found. Sync may still be in progress.
      </p>
    );
  }

  return (
    <div className="border border-zinc-800 rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800 bg-zinc-900/50">
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Repository</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Language</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Branch</th>
            <th className="text-right px-4 py-3 font-medium text-zinc-400">Stars</th>
          </tr>
        </thead>
        <tbody>
          {repos.map((repo) => (
            <tr key={repo.id} className="border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30">
              <td className="px-4 py-3">
                <div>
                  <p className="font-medium text-zinc-100">{repo.name}</p>
                  {repo.description && (
                    <p className="text-zinc-500 text-xs mt-0.5 line-clamp-1">{repo.description}</p>
                  )}
                </div>
              </td>
              <td className="px-4 py-3 text-zinc-400">{repo.language || "—"}</td>
              <td className="px-4 py-3 text-zinc-400">{repo.default_branch}</td>
              <td className="px-4 py-3 text-right text-zinc-400">{repo.stars}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
