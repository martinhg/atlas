import { useState } from "react";
import { useParams, useSearchParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { clearAuth } from "@/lib/auth";
import type { RiskLevel } from "@/lib/api";
import { useImpactAnalysis } from "./useImpactAnalysis";
import { ImpactResultTable } from "./ImpactResultTable";

interface Props {
  onLogout: () => void;
}

const RISK_BADGE_CLASSES: Record<RiskLevel, string> = {
  low: "bg-green-500/10 text-green-400 border-green-500/30",
  medium: "bg-yellow-500/10 text-yellow-400 border-yellow-500/30",
  high: "bg-red-500/10 text-red-400 border-red-500/30",
};

export function ImpactAnalysisPage({ onLogout }: Props) {
  const { slug } = useParams<{ slug: string }>();
  const [searchParams] = useSearchParams();

  const [dependency, setDependency] = useState(searchParams.get("dependency") ?? "");
  const [ecosystem, setEcosystem] = useState(searchParams.get("ecosystem") ?? "npm");

  const { mutate, data, isPending, isError } = useImpactAnalysis(slug!);

  const handleLogout = () => {
    clearAuth();
    onLogout();
  };

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!dependency.trim()) return;
    mutate({ dependency: dependency.trim(), ecosystem });
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
            <span className="text-zinc-300">Impact Analysis</span>
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
          <div>
            <h2 className="text-2xl font-semibold">Impact Analysis</h2>
            <p className="text-zinc-500 text-sm mt-1">
              Find out which repositories and teams are affected by a dependency.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex items-end gap-3">
            <div className="flex-1 space-y-1.5">
              <label htmlFor="impact-dependency" className="text-sm text-zinc-400">
                Dependency name
              </label>
              <Input
                id="impact-dependency"
                placeholder="e.g. lodash"
                value={dependency}
                onChange={(e) => setDependency(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="impact-ecosystem" className="text-sm text-zinc-400">
                Ecosystem
              </label>
              <select
                id="impact-ecosystem"
                value={ecosystem}
                onChange={(e) => setEcosystem(e.target.value)}
                className="flex h-9 rounded-md border border-zinc-800 bg-zinc-950 px-3 py-1 text-sm text-zinc-100 shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-zinc-400"
              >
                <option value="npm">npm</option>
              </select>
            </div>
            <Button type="submit" disabled={isPending || !dependency.trim()}>
              {isPending ? "Analyzing..." : "Analyze"}
            </Button>
          </form>

          {isError && (
            <p className="text-red-400 text-sm">Failed to analyze impact.</p>
          )}

          {data && (
            <div className="space-y-6">
              <div className="flex items-center gap-6">
                <div className="flex items-center gap-2">
                  <span className="text-sm text-zinc-400">Risk:</span>
                  <span
                    className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium ${RISK_BADGE_CLASSES[data.risk_level]}`}
                  >
                    {data.risk_level}
                  </span>
                  <span className="text-sm text-zinc-500">({data.risk_score.toFixed(1)})</span>
                </div>
                <p className="text-sm text-zinc-400">
                  {data.total_repos} repositories affected
                </p>
                <p className="text-sm text-zinc-400">
                  {data.total_teams} teams affected
                </p>
              </div>

              <ImpactResultTable repos={data.affected_repos} />
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
