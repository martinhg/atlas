import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useGraphData } from "@/features/graph/useGraphData";
import type { GraphResponse } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  fetchGraphData: vi.fn(),
}));

import { fetchGraphData } from "@/lib/api";
const mockFetchGraphData = vi.mocked(fetchGraphData);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

const mockGraphResponse: GraphResponse = {
  nodes: [
    { id: "repo:uuid-1", type: "repo", label: "atlas", risk_level: "high" },
    { id: "dep:uuid-2", type: "dep", label: "lodash", ecosystem: "npm", risk_level: "low" },
  ],
  edges: [
    { id: "e1", source: "repo:uuid-1", target: "dep:uuid-2", dep_type: "direct" },
  ],
  truncated: false,
};

describe("useGraphData", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches graph data for a given slug and filters", async () => {
    // Given
    mockFetchGraphData.mockResolvedValue(mockGraphResponse);

    // When
    const { result } = renderHook(
      () => useGraphData("my-org", {}),
      { wrapper: createWrapper() },
    );

    // Then
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockGraphResponse);
    expect(mockFetchGraphData).toHaveBeenCalledWith("my-org", {});
  });

  it("does not call fetchGraphData when slug is empty", async () => {
    // Given
    mockFetchGraphData.mockResolvedValue(mockGraphResponse);

    // When
    renderHook(
      () => useGraphData("", {}),
      { wrapper: createWrapper() },
    );

    // Then — stays pending because enabled:false
    await new Promise((r) => setTimeout(r, 50));
    expect(mockFetchGraphData).not.toHaveBeenCalled();
  });

  it("exposes loading state while fetching", async () => {
    // Given — never resolves immediately, we just check isPending
    let resolve: (v: GraphResponse) => void;
    mockFetchGraphData.mockReturnValue(new Promise((r) => { resolve = r; }));

    // When
    const { result } = renderHook(
      () => useGraphData("my-org", {}),
      { wrapper: createWrapper() },
    );

    // Then
    expect(result.current.isPending).toBe(true);
    resolve!(mockGraphResponse);
  });

  it("exposes error state on fetch failure", async () => {
    // Given
    mockFetchGraphData.mockRejectedValue(new Error("network error"));

    // When
    const { result } = renderHook(
      () => useGraphData("my-org", {}),
      { wrapper: createWrapper() },
    );

    // Then
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("network error");
  });

  it("refetches when filters change", async () => {
    // Given
    mockFetchGraphData.mockResolvedValue(mockGraphResponse);
    const { result, rerender } = renderHook(
      ({ filters }) => useGraphData("my-org", filters),
      { wrapper: createWrapper(), initialProps: { filters: {} } },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockFetchGraphData).toHaveBeenCalledWith("my-org", {});

    // When filters change
    rerender({ filters: { ecosystem: "npm" } });

    // Then new call issued
    await waitFor(() => {
      expect(mockFetchGraphData).toHaveBeenCalledWith("my-org", { ecosystem: "npm" });
    });
  });
});
