import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { GraphNode } from "@/lib/api";
import { NodeDetailPanel } from "@/features/graph/NodeDetailPanel";

const repoNode: GraphNode = {
  id: "repo:uuid-1",
  type: "repo",
  label: "atlas",
  risk_level: "high",
  language: "Go",
};

const depNode: GraphNode = {
  id: "dep:uuid-2",
  type: "dep",
  label: "lodash",
  ecosystem: "npm",
  risk_level: "low",
};

const teamNode: GraphNode = {
  id: "team:backend",
  type: "team",
  label: "@org/backend",
};

describe("NodeDetailPanel", () => {
  const onClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("when a repo node is selected", () => {
    it("displays repo-specific fields", () => {
      // Given / When
      render(<NodeDetailPanel node={repoNode} onClose={onClose} />);

      // Then
      expect(screen.getByText("atlas")).toBeInTheDocument();
      expect(screen.getByText(/repo/i)).toBeInTheDocument();
      expect(screen.getByText(/go/i)).toBeInTheDocument();
      expect(screen.getByText(/high/i)).toBeInTheDocument();
    });
  });

  describe("when a dep node is selected", () => {
    it("displays dep-specific fields including ecosystem", () => {
      // Given / When
      render(<NodeDetailPanel node={depNode} onClose={onClose} />);

      // Then
      expect(screen.getByText("lodash")).toBeInTheDocument();
      expect(screen.getByText(/npm/i)).toBeInTheDocument();
      expect(screen.getByText(/low/i)).toBeInTheDocument();
    });
  });

  describe("when a team node is selected", () => {
    it("displays team-specific fields", () => {
      // Given / When
      render(<NodeDetailPanel node={teamNode} onClose={onClose} />);

      // Then
      expect(screen.getByText("@org/backend")).toBeInTheDocument();
      expect(screen.getByText(/team/i)).toBeInTheDocument();
    });
  });

  it("calls onClose when the close button is clicked", async () => {
    // Given
    const user = userEvent.setup();
    render(<NodeDetailPanel node={repoNode} onClose={onClose} />);

    // When
    await user.click(screen.getByRole("button", { name: /close/i }));

    // Then
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("renders nothing when node is null", () => {
    // Given / When
    const { container } = render(<NodeDetailPanel node={null} onClose={onClose} />);

    // Then
    expect(container.firstChild).toBeNull();
  });
});
