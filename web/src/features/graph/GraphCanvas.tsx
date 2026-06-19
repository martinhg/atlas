import { useEffect, useRef } from "react";
import Graph from "graphology";
import Sigma from "sigma";
import forceAtlas2 from "graphology-layout-forceatlas2";
import type { GraphNode, GraphEdge, RiskLevel } from "@/lib/api";

interface Props {
  nodes: GraphNode[];
  edges: GraphEdge[];
  onSelectNode: (nodeId: string | null) => void;
  className?: string;
}

const NODE_TYPE_COLORS: Record<string, string> = {
  repo: "#6366f1",
  dep: "#f59e0b",
  team: "#22c55e",
};

const RISK_COLORS: Record<RiskLevel, string> = {
  low: "#22c55e",
  medium: "#f59e0b",
  high: "#ef4444",
};

function nodeColor(node: GraphNode): string {
  if (node.risk_level && node.type !== "team") {
    return RISK_COLORS[node.risk_level];
  }
  return NODE_TYPE_COLORS[node.type] ?? "#71717a";
}

function nodeSize(node: GraphNode): number {
  if (node.type === "repo") return 12;
  if (node.type === "team") return 10;
  return 8;
}

export function GraphCanvas({ nodes, edges, onSelectNode, className }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);

  // Keep the latest onSelectNode in a ref so the render effect does NOT depend
  // on its identity. GraphPage passes an inline arrow recreated on every render
  // (e.g. when node-selection state changes), and including it in the effect
  // deps would tear down Sigma and re-run forceAtlas2 on every node click,
  // re-randomizing node positions. The Sigma instance must persist instead.
  const onSelectNodeRef = useRef(onSelectNode);
  useEffect(() => {
    onSelectNodeRef.current = onSelectNode;
  }, [onSelectNode]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const graph = new Graph();

    for (const node of nodes) {
      graph.addNode(node.id, {
        label: node.label,
        color: nodeColor(node),
        size: nodeSize(node),
        x: Math.random(),
        y: Math.random(),
      });
    }

    for (const edge of edges) {
      graph.addEdge(edge.source, edge.target, {
        label: edge.label ?? edge.dep_type,
        color: "#3f3f46",
        size: 1,
      });
    }

    forceAtlas2.assign(graph, { iterations: 50 });

    const sigma = new Sigma(graph, container, {
      renderEdgeLabels: false,
    });

    sigma.on("clickNode", ({ node }) => {
      onSelectNodeRef.current(node);
    });

    sigma.on("clickStage", () => {
      onSelectNodeRef.current(null);
    });

    return () => {
      sigma.kill();
    };
  }, [nodes, edges]);

  return (
    <div
      ref={containerRef}
      data-testid="graph-canvas-container"
      className={className ?? "w-full h-[calc(100vh-64px)]"}
    />
  );
}
