import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import type { GraphNode, GraphEdge } from "@/lib/api";

// Use vi.hoisted so mocks are available before vi.mock factory hoisting
const { mockKill, mockOn, MockSigma, mockAddNode, mockAddEdge, MockGraph, mockAssign } =
  vi.hoisted(() => {
    const mockKill = vi.fn();
    const mockOn = vi.fn();
    const mockSigmaInstance = { kill: mockKill, on: mockOn };
    const MockSigma = vi.fn(() => mockSigmaInstance);

    const mockAddNode = vi.fn();
    const mockAddEdge = vi.fn();
    const mockGraphInstance = { addNode: mockAddNode, addEdge: mockAddEdge };
    const MockGraph = vi.fn(() => mockGraphInstance);

    const mockAssign = vi.fn();

    return { mockKill, mockOn, MockSigma, mockAddNode, mockAddEdge, MockGraph, mockAssign };
  });

vi.mock("sigma", () => ({
  default: MockSigma,
  Sigma: MockSigma,
}));

vi.mock("graphology", () => ({
  default: MockGraph,
  Graph: MockGraph,
}));

vi.mock("graphology-layout-forceatlas2", () => ({
  default: { assign: mockAssign },
}));

import { GraphCanvas } from "@/features/graph/GraphCanvas";

const nodes: GraphNode[] = [
  { id: "repo:uuid-1", type: "repo", label: "atlas", risk_level: "high" },
  { id: "dep:uuid-2", type: "dep", label: "lodash", ecosystem: "npm", risk_level: "low" },
  { id: "team:backend", type: "team", label: "@org/backend" },
];

const edges: GraphEdge[] = [
  { id: "e1", source: "repo:uuid-1", target: "dep:uuid-2", dep_type: "direct" },
  { id: "e2", source: "repo:uuid-1", target: "team:backend", label: "owns" },
];

describe("GraphCanvas", () => {
  const onSelectNode = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders a canvas container in the DOM", () => {
    // Given / When
    render(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={onSelectNode} />,
    );

    // Then
    const container = screen.getByTestId("graph-canvas-container");
    expect(container).toBeInTheDocument();
  });

  it("initializes Sigma with the graph and adds nodes and edges", () => {
    // Given / When
    render(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={onSelectNode} />,
    );

    // Then
    expect(MockGraph).toHaveBeenCalledOnce();
    expect(mockAddNode).toHaveBeenCalledTimes(nodes.length);
    expect(mockAddEdge).toHaveBeenCalledTimes(edges.length);
    expect(MockSigma).toHaveBeenCalledOnce();
  });

  it("applies forceAtlas2 layout after building the graph", () => {
    // Given / When
    render(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={onSelectNode} />,
    );

    // Then
    expect(mockAssign).toHaveBeenCalledOnce();
  });

  it("calls sigma.kill() on unmount — Strict Mode safety", () => {
    // Given
    const { unmount } = render(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={onSelectNode} />,
    );

    // When
    unmount();

    // Then — kill must have been called at least once (cleanup)
    expect(mockKill).toHaveBeenCalled();
  });

  it("registers click handlers for node selection and stage deselect", () => {
    // Given / When
    render(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={onSelectNode} />,
    );

    // Then — sigma.on should have been called for clickNode and clickStage
    const callArgs = mockOn.mock.calls.map(([event]) => event);
    expect(callArgs).toContain("clickNode");
    expect(callArgs).toContain("clickStage");
  });

  it("does NOT recreate the Sigma instance when only the onSelectNode prop identity changes", () => {
    // Given — initial render with one callback identity
    const { rerender } = render(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={() => {}} />,
    );
    expect(MockSigma).toHaveBeenCalledOnce();

    // When — re-render with the SAME nodes/edges but a NEW callback identity
    // (mimics GraphPage passing an inline arrow recreated each render)
    rerender(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={() => {}} />,
    );

    // Then — the effect must NOT tear down and rebuild Sigma. The instance
    // persists across node-selection state changes, so no layout re-randomize.
    expect(MockSigma).toHaveBeenCalledOnce();
    expect(mockKill).not.toHaveBeenCalled();
  });

  it("invokes the latest onSelectNode callback after a re-render without rebuilding Sigma", () => {
    // Given — capture the clickNode handler registered with sigma.on
    const first = vi.fn();
    const { rerender } = render(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={first} />,
    );
    const clickNodeHandler = mockOn.mock.calls.find(
      ([event]) => event === "clickNode",
    )?.[1] as (payload: { node: string }) => void;

    // When — the parent re-renders with a brand-new callback identity
    const second = vi.fn();
    rerender(
      <GraphCanvas nodes={nodes} edges={edges} onSelectNode={second} />,
    );
    // And the original Sigma click handler fires
    clickNodeHandler({ node: "repo:uuid-1" });

    // Then — the LATEST callback receives the event, not the stale one
    expect(second).toHaveBeenCalledWith("repo:uuid-1");
    expect(first).not.toHaveBeenCalled();
    expect(MockSigma).toHaveBeenCalledOnce();
  });
});
