import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { clearAuth } from "@/lib/auth";
import type { GraphFilters, GraphNode } from "@/lib/api";
import { useGraphData } from "./useGraphData";
import { GraphCanvas } from "./GraphCanvas";
import { GraphFilters as GraphFiltersComponent } from "./GraphFilters";
import { NodeDetailPanel } from "./NodeDetailPanel";

interface Props {
  onLogout: () => void;
}

export default function GraphPage({ onLogout }: Props) {
  const { slug } = useParams<{ slug: string }>();
  const [filters, setFilters] = useState<GraphFilters>({});
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);

  const { data, isPending, isError, refetch } = useGraphData(slug ?? "", filters);

  const handleLogout = () => {
    clearAuth();
    onLogout();
  };

  const hasTeamNodes = (data?.nodes ?? []).some((n) => n.type === "team");

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 flex flex-col">
      <header className="border-b border-zinc-800 px-6 py-4 flex-shrink-0">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link
              to="/dashboard"
              className="text-xl font-bold tracking-tight hover:text-zinc-300"
            >
              Atlas
            </Link>
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-300">Dependency Graph</span>
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

      <GraphFiltersComponent filters={filters} onChange={setFilters} />

      {/* Status banners */}
      {data?.truncated && (
        <div className="bg-yellow-500/10 border-b border-yellow-500/30 px-6 py-2 text-sm text-yellow-300">
          Showing a partial view of the graph — more than 5 000 edges were found. Apply filters to
          narrow the result.
        </div>
      )}
      {data && !isPending && !hasTeamNodes && (
        <div className="bg-zinc-800/50 border-b border-zinc-700 px-6 py-2 text-sm text-zinc-400">
          No team ownership data synced yet.
        </div>
      )}

      {/* Main content */}
      <main className="flex-1 relative">
        {isPending && (
          <div className="flex items-center justify-center h-full py-24">
            <p className="text-zinc-500 animate-pulse">Loading graph data...</p>
          </div>
        )}

        {isError && (
          <div className="flex flex-col items-center justify-center h-full py-24 gap-4">
            <p className="text-red-400">Failed to load graph data.</p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              className="border-zinc-700 text-zinc-400 hover:bg-zinc-800"
            >
              Retry
            </Button>
          </div>
        )}

        {!isPending && !isError && data && data.nodes.length === 0 && (
          <div className="flex items-center justify-center h-full py-24">
            <p className="text-zinc-500">No graph data for this org yet.</p>
          </div>
        )}

        {!isPending && !isError && data && data.nodes.length > 0 && (
          <div className="relative h-full">
            <GraphCanvas
              nodes={data.nodes}
              edges={data.edges}
              onSelectNode={(id) => {
                if (id === null) {
                  setSelectedNode(null);
                  return;
                }
                const node = data.nodes.find((n) => n.id === id) ?? null;
                setSelectedNode(node);
              }}
              className="w-full h-[calc(100vh-64px)]"
            />
            {selectedNode && (
              <NodeDetailPanel
                node={selectedNode}
                onClose={() => setSelectedNode(null)}
              />
            )}
          </div>
        )}
      </main>
    </div>
  );
}
