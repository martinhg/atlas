import { cn } from "@/lib/utils";
import type { DependencyWithCount } from "@/lib/api";

interface Props {
  deps: DependencyWithCount[];
  onRowClick?: (dep: DependencyWithCount) => void;
}

export function DependencyTable({ deps, onRowClick }: Props) {
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
            <th className="text-right px-4 py-3 font-medium text-zinc-400">Repos</th>
          </tr>
        </thead>
        <tbody>
          {deps.map((dep) => (
            <tr
              key={`${dep.ecosystem}/${dep.name}`}
              className={cn(
                "border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30",
                onRowClick && "cursor-pointer",
              )}
              onClick={onRowClick ? () => onRowClick(dep) : undefined}
              {...(onRowClick && {
                tabIndex: 0,
                role: "link" as const,
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
              <td className="px-4 py-3 text-right text-zinc-400">{dep.repo_count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
