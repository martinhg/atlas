import { Link } from "react-router-dom";
import type { RepoOwnerSummary } from "@/lib/api";

interface Props {
  data: RepoOwnerSummary[];
  slug: string;
}

export function OwnershipTable({ data, slug }: Props) {
  if (data.length === 0) {
    return (
      <p className="text-zinc-500 text-sm py-8 text-center">
        No ownership data found.
      </p>
    );
  }

  return (
    <div className="border border-zinc-800 rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800 bg-zinc-900/50">
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Repository</th>
            <th className="text-right px-4 py-3 font-medium text-zinc-400">Owners</th>
            <th className="text-right px-4 py-3 font-medium text-zinc-400">Teams</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Team Names</th>
          </tr>
        </thead>
        <tbody>
          {data.map((summary) => (
            <tr
              key={summary.repo_name}
              className="border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30"
            >
              <td className="px-4 py-3 font-medium text-zinc-100">
                <Link
                  to={`/orgs/${slug}/ownership/${summary.repo_name}`}
                  className="hover:text-zinc-300 underline-offset-2 hover:underline"
                >
                  {summary.repo_name}
                </Link>
              </td>
              <td className="px-4 py-3 text-right text-zinc-400">{summary.owner_count}</td>
              <td className="px-4 py-3 text-right text-zinc-400">{summary.team_count}</td>
              <td className="px-4 py-3 text-zinc-400">
                {summary.teams.length === 0 ? (
                  <span className="text-zinc-600">—</span>
                ) : (
                  <span className="flex flex-wrap gap-1">
                    {summary.teams.map((team) => (
                      <span
                        key={team}
                        className="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium bg-zinc-800 text-zinc-300 border border-zinc-700"
                      >
                        {team}
                      </span>
                    ))}
                  </span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
