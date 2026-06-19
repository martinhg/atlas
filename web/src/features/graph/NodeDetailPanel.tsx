import { useEffect, useMemo, useRef } from "react";
import type { GraphEdge, GraphNode, RiskLevel } from "@/lib/api";
import { cn } from "@/lib/utils";

interface Props {
  node: GraphNode | null;
  /** Full node set, used to resolve edge endpoints into labels. */
  nodes: GraphNode[];
  /** Full edge set, used to derive relationships client-side. */
  edges: GraphEdge[];
  onClose: () => void;
  className?: string;
}

const RISK_BADGE_CLASSES = {
  low: "bg-green-500/10 text-green-400 border-green-500/30",
  medium: "bg-yellow-500/10 text-yellow-400 border-yellow-500/30",
  high: "bg-red-500/10 text-red-400 border-red-500/30",
} as const;

const RISK_ORDER: Record<RiskLevel, number> = { low: 0, medium: 1, high: 2 };
const TOP_DEPS_LIMIT = 5;

function RiskBadge({ level }: { level: RiskLevel }) {
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

function TypeBadge({ type }: { type: string }) {
  return (
    <span className="text-xs uppercase tracking-wider text-zinc-500 border border-zinc-700 rounded px-1.5 py-0.5">
      {type}
    </span>
  );
}

function isType(prefix: string, id: string) {
  return id.startsWith(`${prefix}:`);
}

function RepoDetail({
  node,
  nodes,
  edges,
}: {
  node: GraphNode;
  nodes: GraphNode[];
  edges: GraphEdge[];
}) {
  const nodeById = useMemo(
    () => new Map(nodes.map((n) => [n.id, n])),
    [nodes],
  );

  // Owning teams: repo→team edges where source == this repo.
  const owningTeams = useMemo(
    () =>
      edges
        .filter((e) => e.source === node.id && isType("team", e.target))
        .map((e) => nodeById.get(e.target))
        .filter((n): n is GraphNode => Boolean(n)),
    [edges, node.id, nodeById],
  );

  // Top deps by risk: repo→dep edges where source == this repo, sorted desc.
  const topDeps = useMemo(
    () =>
      edges
        .filter((e) => e.source === node.id && isType("dep", e.target))
        .map((e) => nodeById.get(e.target))
        .filter((n): n is GraphNode => Boolean(n))
        .sort(
          (a, b) =>
            RISK_ORDER[b.risk_level ?? "low"] -
            RISK_ORDER[a.risk_level ?? "low"],
        )
        .slice(0, TOP_DEPS_LIMIT),
    [edges, node.id, nodeById],
  );

  return (
    <>
      <div className="flex items-center gap-2 flex-wrap">
        <TypeBadge type="repo" />
        {node.risk_level && <RiskBadge level={node.risk_level} />}
      </div>
      <h3 className="mt-2 text-lg font-semibold text-zinc-100">{node.label}</h3>
      {node.language && (
        <p className="mt-1 text-sm text-zinc-400">
          Language: <span className="text-zinc-200">{node.language}</span>
        </p>
      )}

      <div className="mt-4">
        <p className="text-xs uppercase tracking-wider text-zinc-500">
          Owning teams
        </p>
        {owningTeams.length > 0 ? (
          <ul className="mt-1 space-y-0.5">
            {owningTeams.map((team) => (
              <li key={team.id} className="text-sm text-zinc-200">
                {team.label}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-1 text-sm text-zinc-500">None</p>
        )}
      </div>

      <div className="mt-4">
        <p className="text-xs uppercase tracking-wider text-zinc-500">
          Top deps by risk
        </p>
        {topDeps.length > 0 ? (
          <ul className="mt-1 space-y-1">
            {topDeps.map((dep) => (
              <li
                key={dep.id}
                className="flex items-center justify-between gap-2 text-sm text-zinc-200"
              >
                <span className="truncate">{dep.label}</span>
                {dep.risk_level && <RiskBadge level={dep.risk_level} />}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-1 text-sm text-zinc-500">None</p>
        )}
      </div>
    </>
  );
}

function DepDetail({
  node,
  edges,
}: {
  node: GraphNode;
  edges: GraphEdge[];
}) {
  // Affected repos: number of repo→dep edges whose target is this dep.
  const affectedRepos = useMemo(
    () =>
      edges.filter((e) => e.target === node.id && isType("repo", e.source))
        .length,
    [edges, node.id],
  );

  return (
    <>
      <div className="flex items-center gap-2 flex-wrap">
        <TypeBadge type="dep" />
        {node.risk_level && <RiskBadge level={node.risk_level} />}
      </div>
      <h3 className="mt-2 text-lg font-semibold text-zinc-100">{node.label}</h3>
      {node.ecosystem && (
        <p className="mt-1 text-sm text-zinc-400">
          Ecosystem: <span className="text-zinc-200">{node.ecosystem}</span>
        </p>
      )}
      <p className="mt-1 text-sm text-zinc-400">
        Affected repos: <span className="text-zinc-200">{affectedRepos}</span>
      </p>
      {/*
        Vuln count is intentionally descoped for this PR: the graph endpoint
        payload does not yet expose per-dep vulnerability counts (Epic 8 vuln
        severity is not plumbed into the graph response). Rendered as "—" rather
        than fabricated. Deferred follow-up: add a `vuln_count` field to the
        graph dep-node payload, then surface it here.
      */}
      <p className="mt-1 text-sm text-zinc-400">
        Vuln count: <span className="text-zinc-200">—</span>
      </p>
    </>
  );
}

function TeamDetail({
  node,
  nodes,
  edges,
}: {
  node: GraphNode;
  nodes: GraphNode[];
  edges: GraphEdge[];
}) {
  const nodeById = useMemo(
    () => new Map(nodes.map((n) => [n.id, n])),
    [nodes],
  );

  // Repos owned: repo→team edges where target == this team.
  const ownedRepos = useMemo(
    () =>
      edges
        .filter((e) => e.target === node.id && isType("repo", e.source))
        .map((e) => nodeById.get(e.source))
        .filter((n): n is GraphNode => Boolean(n)),
    [edges, node.id, nodeById],
  );

  return (
    <>
      <div className="flex items-center gap-2">
        <TypeBadge type="team" />
      </div>
      <h3 className="mt-2 text-lg font-semibold text-zinc-100">{node.label}</h3>

      <div className="mt-4">
        <p className="text-xs uppercase tracking-wider text-zinc-500">
          Repos owned
        </p>
        {ownedRepos.length > 0 ? (
          <ul className="mt-1 space-y-0.5">
            {ownedRepos.map((repo) => (
              <li key={repo.id} className="text-sm text-zinc-200">
                {repo.label}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-1 text-sm text-zinc-500">None</p>
        )}
      </div>
    </>
  );
}

export function NodeDetailPanel({
  node,
  nodes,
  edges,
  onClose,
  className,
}: Props) {
  const panelRef = useRef<HTMLElement>(null);

  // Spec: clicking outside the panel (or pressing Esc) MUST close it.
  useEffect(() => {
    if (!node) return;

    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (panelRef.current && target && !panelRef.current.contains(target)) {
        onClose();
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [node, onClose]);

  if (!node) return null;

  return (
    <aside
      ref={panelRef}
      className={cn(
        "absolute right-0 top-0 h-full w-72 bg-zinc-900 border-l border-zinc-800 p-4 z-10 overflow-y-auto",
        className,
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          {node.type === "repo" && (
            <RepoDetail node={node} nodes={nodes} edges={edges} />
          )}
          {node.type === "dep" && <DepDetail node={node} edges={edges} />}
          {node.type === "team" && (
            <TeamDetail node={node} nodes={nodes} edges={edges} />
          )}
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
