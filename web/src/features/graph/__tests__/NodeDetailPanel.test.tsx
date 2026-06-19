import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { GraphNode, GraphEdge } from "@/lib/api";
import { NodeDetailPanel } from "@/features/graph/NodeDetailPanel";

// Fixtures: one repo owned by one team, depending on two deps of differing risk.
const repoNode: GraphNode = {
  id: "repo:uuid-1",
  type: "repo",
  label: "atlas",
  risk_level: "high",
  language: "Go",
};

const otherRepoNode: GraphNode = {
  id: "repo:uuid-9",
  type: "repo",
  label: "atlas-web",
  risk_level: "low",
  language: "TypeScript",
};

const depHigh: GraphNode = {
  id: "dep:uuid-2",
  type: "dep",
  label: "lodash",
  ecosystem: "npm",
  risk_level: "high",
};

const depLow: GraphNode = {
  id: "dep:uuid-3",
  type: "dep",
  label: "left-pad",
  ecosystem: "npm",
  risk_level: "low",
};

const teamNode: GraphNode = {
  id: "team:backend",
  type: "team",
  label: "@org/backend",
};

const nodes: GraphNode[] = [
  repoNode,
  otherRepoNode,
  depHigh,
  depLow,
  teamNode,
];

const edges: GraphEdge[] = [
  // repo:uuid-1 depends on both deps
  { id: "e1", source: "repo:uuid-1", target: "dep:uuid-2", dep_type: "direct" },
  { id: "e2", source: "repo:uuid-1", target: "dep:uuid-3", dep_type: "direct" },
  // repo:uuid-9 depends on the high-risk dep too (so it has 2 affected repos)
  { id: "e3", source: "repo:uuid-9", target: "dep:uuid-2", dep_type: "direct" },
  // both repos owned by @org/backend
  { id: "e4", source: "repo:uuid-1", target: "team:backend", label: "owns" },
  { id: "e5", source: "repo:uuid-9", target: "team:backend", label: "owns" },
];

describe("NodeDetailPanel", () => {
  const onClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("when a repo node is selected", () => {
    it("displays repo-specific fields including owning teams and top deps", () => {
      // Given / When
      render(
        <NodeDetailPanel
          node={repoNode}
          nodes={nodes}
          edges={edges}
          onClose={onClose}
        />,
      );

      // Then — base fields
      expect(screen.getByText("atlas")).toBeInTheDocument();
      expect(screen.getByText(/repo/i)).toBeInTheDocument();
      expect(screen.getByText(/go/i)).toBeInTheDocument();
      // "high" appears for the repo's own risk badge and the high-risk dep badge
      expect(screen.getAllByText(/high/i).length).toBeGreaterThan(0);

      // And — owning teams derived from repo→team edges
      expect(screen.getByText(/owning teams/i)).toBeInTheDocument();
      expect(screen.getByText("@org/backend")).toBeInTheDocument();

      // And — top deps by risk derived from repo→dep edges
      expect(screen.getByText(/top deps/i)).toBeInTheDocument();
      expect(screen.getByText("lodash")).toBeInTheDocument();
      expect(screen.getByText("left-pad")).toBeInTheDocument();
    });
  });

  describe("when a dep node is selected", () => {
    it("displays dep fields including affected-repos count derived from edges", () => {
      // Given / When — depHigh is depended on by two repos
      render(
        <NodeDetailPanel
          node={depHigh}
          nodes={nodes}
          edges={edges}
          onClose={onClose}
        />,
      );

      // Then
      expect(screen.getByText("lodash")).toBeInTheDocument();
      expect(screen.getByText(/npm/i)).toBeInTheDocument();
      expect(screen.getByText(/high/i)).toBeInTheDocument();

      // And — affected repos count (2 repo→dep edges target this dep)
      expect(screen.getByText(/affected repos/i)).toBeInTheDocument();
      expect(screen.getByText("2")).toBeInTheDocument();
    });

    it("renders vuln count as a deferred placeholder, not a fabricated number", () => {
      // Given / When
      render(
        <NodeDetailPanel
          node={depHigh}
          nodes={nodes}
          edges={edges}
          onClose={onClose}
        />,
      );

      // Then — vuln count is descoped: shown as a placeholder dash
      const vulnRow = screen.getByText(/vuln count/i).closest("p");
      expect(vulnRow).toHaveTextContent("—");
    });
  });

  describe("when a team node is selected", () => {
    it("displays team fields including the list of owned repos", () => {
      // Given / When
      render(
        <NodeDetailPanel
          node={teamNode}
          nodes={nodes}
          edges={edges}
          onClose={onClose}
        />,
      );

      // Then
      expect(screen.getByText("@org/backend")).toBeInTheDocument();
      expect(screen.getByText(/team/i)).toBeInTheDocument();

      // And — owned repos derived from repo→team edges
      expect(screen.getByText(/repos owned/i)).toBeInTheDocument();
      expect(screen.getByText("atlas")).toBeInTheDocument();
      expect(screen.getByText("atlas-web")).toBeInTheDocument();
    });
  });

  it("calls onClose when the close button is clicked", async () => {
    // Given
    const user = userEvent.setup();
    render(
      <NodeDetailPanel
        node={repoNode}
        nodes={nodes}
        edges={edges}
        onClose={onClose}
      />,
    );

    // When
    await user.click(screen.getByRole("button", { name: /close/i }));

    // Then
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose when clicking outside the panel", async () => {
    // Given
    const user = userEvent.setup();
    render(
      <div>
        <button type="button">elsewhere</button>
        <NodeDetailPanel
          node={repoNode}
          nodes={nodes}
          edges={edges}
          onClose={onClose}
        />
      </div>,
    );

    // When — click an element outside the panel
    await user.click(screen.getByRole("button", { name: /elsewhere/i }));

    // Then
    expect(onClose).toHaveBeenCalled();
  });

  it("does NOT call onClose when clicking inside the panel", async () => {
    // Given
    const user = userEvent.setup();
    render(
      <NodeDetailPanel
        node={repoNode}
        nodes={nodes}
        edges={edges}
        onClose={onClose}
      />,
    );

    // When — click the heading inside the panel
    await user.click(screen.getByText("atlas"));

    // Then
    expect(onClose).not.toHaveBeenCalled();
  });

  it("calls onClose when Escape is pressed", async () => {
    // Given
    const user = userEvent.setup();
    render(
      <NodeDetailPanel
        node={repoNode}
        nodes={nodes}
        edges={edges}
        onClose={onClose}
      />,
    );

    // When
    await user.keyboard("{Escape}");

    // Then
    expect(onClose).toHaveBeenCalled();
  });

  it("renders nothing when node is null", () => {
    // Given / When
    const { container } = render(
      <NodeDetailPanel node={null} nodes={nodes} edges={edges} onClose={onClose} />,
    );

    // Then
    expect(container.firstChild).toBeNull();
  });
});
