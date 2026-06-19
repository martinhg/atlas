import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { GraphResponse } from "@/lib/api";

// Mock useGraphData so we control what the page receives
vi.mock("@/features/graph/useGraphData", () => ({
  useGraphData: vi.fn(),
}));

// Mock GraphCanvas to avoid sigma/graphology in jsdom
const { mockGraphCanvas } = vi.hoisted(() => {
  const mockGraphCanvas = vi.fn(() => <div data-testid="graph-canvas" />);
  return { mockGraphCanvas };
});

vi.mock("@/features/graph/GraphCanvas", () => ({
  GraphCanvas: mockGraphCanvas,
}));

import { useGraphData } from "@/features/graph/useGraphData";
import GraphPage from "@/features/graph/GraphPage";

const mockUseGraphData = vi.mocked(useGraphData);

const FULL_GRAPH: GraphResponse = {
  nodes: [
    { id: "repo:uuid-1", type: "repo", label: "atlas", risk_level: "high" },
    { id: "dep:uuid-2", type: "dep", label: "lodash", ecosystem: "npm", risk_level: "low" },
    { id: "team:backend", type: "team", label: "@org/backend" },
  ],
  edges: [
    { id: "e1", source: "repo:uuid-1", target: "dep:uuid-2", dep_type: "direct" },
    { id: "e2", source: "repo:uuid-1", target: "team:backend", label: "owns" },
  ],
  truncated: false,
};

const NO_TEAM_GRAPH: GraphResponse = {
  nodes: [
    { id: "repo:uuid-1", type: "repo", label: "atlas", risk_level: "low" },
    { id: "dep:uuid-2", type: "dep", label: "react", ecosystem: "npm", risk_level: "low" },
  ],
  edges: [
    { id: "e1", source: "repo:uuid-1", target: "dep:uuid-2", dep_type: "direct" },
  ],
  truncated: false,
};

const TRUNCATED_GRAPH: GraphResponse = {
  ...FULL_GRAPH,
  truncated: true,
};

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/orgs/my-org/graph"]}>
        <Routes>
          <Route path="/orgs/:slug/graph" element={<GraphPage onLogout={vi.fn()} />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("GraphPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows a loading indicator while fetching and does not render the canvas", () => {
    // Given — query is pending
    mockUseGraphData.mockReturnValue({
      isPending: true,
      isError: false,
      data: undefined,
      error: null,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useGraphData>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
    expect(screen.queryByTestId("graph-canvas")).not.toBeInTheDocument();
  });

  it("shows an error message and retry button on fetch failure", async () => {
    // Given
    const refetch = vi.fn();
    mockUseGraphData.mockReturnValue({
      isPending: false,
      isError: true,
      data: undefined,
      error: new Error("network error"),
      refetch,
    } as unknown as ReturnType<typeof useGraphData>);

    // When
    const user = userEvent.setup();
    renderPage();

    // Then
    expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    const retryButton = screen.getByRole("button", { name: /retry/i });
    expect(retryButton).toBeInTheDocument();

    // And retry triggers refetch
    await user.click(retryButton);
    expect(refetch).toHaveBeenCalledOnce();
  });

  it("shows empty-state message when nodes array is empty", () => {
    // Given
    mockUseGraphData.mockReturnValue({
      isPending: false,
      isError: false,
      data: { nodes: [], edges: [], truncated: false },
      error: null,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useGraphData>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/no graph data/i)).toBeInTheDocument();
    expect(screen.queryByTestId("graph-canvas")).not.toBeInTheDocument();
  });

  it("renders the canvas when data is present", () => {
    // Given
    mockUseGraphData.mockReturnValue({
      isPending: false,
      isError: false,
      data: FULL_GRAPH,
      error: null,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useGraphData>);

    // When
    renderPage();

    // Then
    expect(screen.getByTestId("graph-canvas")).toBeInTheDocument();
  });

  it("shows a truncated banner when graph is partial", () => {
    // Given
    mockUseGraphData.mockReturnValue({
      isPending: false,
      isError: false,
      data: TRUNCATED_GRAPH,
      error: null,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useGraphData>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/partial/i)).toBeInTheDocument();
  });

  it("shows a no-team notice when graph has no team nodes", () => {
    // Given
    mockUseGraphData.mockReturnValue({
      isPending: false,
      isError: false,
      data: NO_TEAM_GRAPH,
      error: null,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useGraphData>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/no team ownership/i)).toBeInTheDocument();
  });

  it("re-calls useGraphData with updated filters when a filter changes", async () => {
    // Given
    mockUseGraphData.mockReturnValue({
      isPending: false,
      isError: false,
      data: FULL_GRAPH,
      error: null,
      refetch: vi.fn(),
    } as unknown as ReturnType<typeof useGraphData>);

    const user = userEvent.setup();
    renderPage();

    // When — change ecosystem filter
    const ecosystemSelect = screen.getByRole("combobox", { name: /ecosystem/i });
    await user.selectOptions(ecosystemSelect, "npm");

    // Then — useGraphData should have been called again with the new filter
    await waitFor(() => {
      const calls = mockUseGraphData.mock.calls;
      const callsWithEcosystem = calls.filter(
        ([, filters]) => filters?.ecosystem === "npm",
      );
      expect(callsWithEcosystem.length).toBeGreaterThan(0);
    });
  });
});
