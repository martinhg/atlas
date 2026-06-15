import { cn } from "@/lib/utils";
import type { OwnerRule } from "@/lib/api";

interface Props {
  rules: OwnerRule[];
}

function ownerTypeBadgeClass(ownerType: string): string {
  switch (ownerType) {
    case "team":
      return "bg-blue-900/40 text-blue-300 border-blue-700/50";
    case "email":
      return "bg-zinc-800 text-zinc-400 border-zinc-700";
    default:
      // user
      return "bg-zinc-800 text-zinc-300 border-zinc-700";
  }
}

export function OwnershipDetailTable({ rules }: Props) {
  if (rules.length === 0) {
    return (
      <p className="text-zinc-500 text-sm py-8 text-center">
        No CODEOWNERS rules found for this repository.
      </p>
    );
  }

  return (
    <div className="border border-zinc-800 rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-zinc-800 bg-zinc-900/50">
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Pattern</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Owner</th>
            <th className="text-left px-4 py-3 font-medium text-zinc-400">Type</th>
            <th className="text-right px-4 py-3 font-medium text-zinc-400">Line</th>
          </tr>
        </thead>
        <tbody>
          {rules.map((rule, index) => (
            <tr
              key={`${rule.pattern}-${rule.owner}-${index}`}
              className="border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30"
            >
              <td className="px-4 py-3 font-mono text-zinc-100">{rule.pattern}</td>
              <td className="px-4 py-3 text-zinc-400">{rule.owner}</td>
              <td className="px-4 py-3">
                <span
                  className={cn(
                    "inline-flex items-center rounded px-1.5 py-0.5 text-xs font-medium border",
                    ownerTypeBadgeClass(rule.owner_type),
                  )}
                >
                  {rule.owner_type}
                </span>
              </td>
              <td className="px-4 py-3 text-right text-zinc-400">
                {rule.line_number != null ? rule.line_number : <span className="text-zinc-600">—</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
