import type { GraphNode } from "@/lib/api";
import { cn } from "@/lib/utils";

interface Props {
  node: GraphNode | null;
  onClose: () => void;
  className?: string;
}

const RISK_BADGE_CLASSES = {
  low: "bg-green-500/10 text-green-400 border-green-500/30",
  medium: "bg-yellow-500/10 text-yellow-400 border-yellow-500/30",
  high: "bg-red-500/10 text-red-400 border-red-500/30",
} as const;

function RiskBadge({ level }: { level: "low" | "medium" | "high" }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium",
        RISK_BADGE_CLASSES[level],
      )}
    >
      {level}
    </span>
  );
}

function RepoDetail({ node }: { node: GraphNode }) {
  return (
    <>
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-xs uppercase tracking-wider text-zinc-500 border border-zinc-700 rounded px-1.5 py-0.5">
          repo
        </span>
        {node.risk_level && <RiskBadge level={node.risk_level} />}
      </div>
      <h3 className="mt-2 text-lg font-semibold text-zinc-100">{node.label}</h3>
      {node.language && (
        <p className="mt-1 text-sm text-zinc-400">
          Language: <span className="text-zinc-200">{node.language}</span>
        </p>
      )}
    </>
  );
}

function DepDetail({ node }: { node: GraphNode }) {
  return (
    <>
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-xs uppercase tracking-wider text-zinc-500 border border-zinc-700 rounded px-1.5 py-0.5">
          dep
        </span>
        {node.risk_level && <RiskBadge level={node.risk_level} />}
      </div>
      <h3 className="mt-2 text-lg font-semibold text-zinc-100">{node.label}</h3>
      {node.ecosystem && (
        <p className="mt-1 text-sm text-zinc-400">
          Ecosystem: <span className="text-zinc-200">{node.ecosystem}</span>
        </p>
      )}
    </>
  );
}

function TeamDetail({ node }: { node: GraphNode }) {
  return (
    <>
      <div className="flex items-center gap-2">
        <span className="text-xs uppercase tracking-wider text-zinc-500 border border-zinc-700 rounded px-1.5 py-0.5">
          team
        </span>
      </div>
      <h3 className="mt-2 text-lg font-semibold text-zinc-100">{node.label}</h3>
    </>
  );
}

export function NodeDetailPanel({ node, onClose, className }: Props) {
  if (!node) return null;

  return (
    <aside
      className={cn(
        "absolute right-0 top-0 h-full w-72 bg-zinc-900 border-l border-zinc-800 p-4 z-10 overflow-y-auto",
        className,
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          {node.type === "repo" && <RepoDetail node={node} />}
          {node.type === "dep" && <DepDetail node={node} />}
          {node.type === "team" && <TeamDetail node={node} />}
        </div>
        <button
          type="button"
          aria-label="close"
          onClick={onClose}
          className="ml-2 flex-shrink-0 text-zinc-500 hover:text-zinc-300"
        >
          ✕
        </button>
      </div>
    </aside>
  );
}
