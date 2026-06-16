import type { ImpactAffectedRepo } from "@/lib/api";

interface Props {
  repos: ImpactAffectedRepo[];
}

export function ImpactResultTable({ repos }: Props) {
  if (repos.length === 0) {
    return (
      <p className="text-zinc-500 text-sm py-8 text-center">
        No repositories are affected by this dependency.
      </p>
    );
  }

  return (
    <div className="border border-zinc-800 rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800 bg-zinc-900/50">
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Repository</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Version</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Type</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Owners</th>
          </tr>
        </thead>
        <tbody>
          {repos.map((repo) => (
            <tr
              key={repo.id}
              className="border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30"
            >
              <td className="px-4 py-3 font-medium text-zinc-100">{repo.full_name}</td>
              <td className="px-4 py-3 text-zinc-400">{repo.version}</td>
              <td className="px-4 py-3 text-zinc-400">{repo.dep_type}</td>
              <td className="px-4 py-3 text-zinc-400">
                {repo.teams.length > 0 ? repo.teams.join(", ") : "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
