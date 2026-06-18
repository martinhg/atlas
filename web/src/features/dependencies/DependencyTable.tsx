import { useParams, useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import type { DependencyWithCount, SeverityLevel } from "@/lib/api";
import { SeverityBadge } from "@/features/vulnerabilities/SeverityBadge";

interface Props {
  deps: DependencyWithCount[];
  onRowClick?: (dep: DependencyWithCount) => void;
}

export function DependencyTable({ deps, onRowClick }: Props) {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();

  if (deps.length === 0) {
    return (
      <p className="text-zinc-500 text-sm py-8 text-center">
        No dependencies found.
      </p>
    );
  }

  return (
    <div className="border border-zinc-800 rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800 bg-zinc-900/50">
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Name</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Ecosystem</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Vulnerabilities</th>
            <th className="text-right px-4 py-3 font-medium text-zinc-400">Repos</th>
          </tr>
        </thead>
        <tbody>
          {deps.map((dep) => {
            const vulnCount = dep.vuln_count ?? 0;
            return (
              <tr
                key={`${dep.ecosystem}/${dep.name}`}
                className={cn(
                  "border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30",
                  onRowClick && "cursor-pointer",
                )}
                onClick={onRowClick ? () => onRowClick(dep) : undefined}
                {...(onRowClick && {
                  tabIndex: 0,
                  "aria-label": `View ${dep.name} dependency details`,
                  onKeyDown: (e: React.KeyboardEvent<HTMLTableRowElement>) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onRowClick(dep);
                    }
                  },
                })}
              >
                <td className="px-4 py-3 font-medium text-zinc-100">{dep.name}</td>
                <td className="px-4 py-3 text-zinc-400">{dep.ecosystem}</td>
                <td className="px-4 py-3">
                  {vulnCount > 0 ? (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        navigate(
                          `/orgs/${slug}/vulnerabilities?package=${encodeURIComponent(dep.name)}`,
                        );
                      }}
                      aria-label={`View ${vulnCount} vulnerabilities for ${dep.name}`}
                      className="inline-flex items-center gap-2 hover:opacity-80"
                    >
                      <span className="text-zinc-200">{vulnCount}</span>
                      {dep.max_severity && (
                        <SeverityBadge severity={dep.max_severity as SeverityLevel} />
                      )}
                    </button>
                  ) : (
                    <span className="text-zinc-600">0</span>
                  )}
                </td>
                <td className="px-4 py-3 text-right text-zinc-400">{dep.repo_count}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
