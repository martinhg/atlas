import type { GraphFilters, RiskLevel } from "@/lib/api";
import { cn } from "@/lib/utils";

const ECOSYSTEMS = ["", "npm", "pypi", "go", "maven", "cargo", "rubygems"];
const RISK_LEVELS: RiskLevel[] = ["low", "medium", "high"];

interface Props {
  filters: GraphFilters;
  onChange: (filters: GraphFilters) => void;
  className?: string;
}

export function GraphFilters({ filters, onChange, className }: Props) {
  const handleEcosystemChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    onChange({ ...filters, ecosystem: value || undefined });
  };

  const handleRiskChange = (level: RiskLevel, checked: boolean) => {
    if (checked) {
      onChange({ ...filters, risk: level });
    } else {
      const next = { ...filters };
      delete next.risk;
      onChange(next);
    }
  };

  const handleTeamChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    onChange({ ...filters, team: value || undefined });
  };

  return (
    <div
      className={cn(
        "flex flex-wrap items-end gap-4 px-4 py-3 border-b border-zinc-800 bg-zinc-900",
        className,
      )}
    >
      {/* Ecosystem filter */}
      <div className="flex flex-col gap-1">
        <label
          htmlFor="graph-filter-ecosystem"
          className="text-xs text-zinc-400"
        >
          Ecosystem
        </label>
        <select
          id="graph-filter-ecosystem"
          aria-label="ecosystem"
          value={filters.ecosystem ?? ""}
          onChange={handleEcosystemChange}
          className="flex h-8 rounded-md border border-zinc-700 bg-zinc-950 px-2 py-0.5 text-sm text-zinc-100 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-zinc-400"
        >
          <option value="">All ecosystems</option>
          {ECOSYSTEMS.filter(Boolean).map((eco) => (
            <option key={eco} value={eco}>
              {eco}
            </option>
          ))}
        </select>
      </div>

      {/* Risk filter */}
      <div className="flex flex-col gap-1">
        <span className="text-xs text-zinc-400">Risk</span>
        <div className="flex items-center gap-3">
          {RISK_LEVELS.map((level) => (
            <label
              key={level}
              className="flex items-center gap-1.5 text-sm text-zinc-300 cursor-pointer"
            >
              <input
                type="checkbox"
                aria-label={level}
                checked={filters.risk === level}
                onChange={(e) => handleRiskChange(level, e.target.checked)}
                className="h-4 w-4 rounded border-zinc-600 bg-zinc-800 accent-indigo-500"
              />
              {level}
            </label>
          ))}
        </div>
      </div>

      {/* Team filter */}
      <div className="flex flex-col gap-1">
        <label htmlFor="graph-filter-team" className="text-xs text-zinc-400">
          Team
        </label>
        <input
          id="graph-filter-team"
          type="text"
          aria-label="team"
          placeholder="@org/team"
          value={filters.team ?? ""}
          onChange={handleTeamChange}
          className="flex h-8 rounded-md border border-zinc-700 bg-zinc-950 px-2 py-0.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-zinc-400"
        />
      </div>
    </div>
  );
}
