import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useOwnership } from "@/features/ownership/useOwnership";

vi.mock("@/lib/api", () => ({
  fetchOwnership: vi.fn(),
}));

import { fetchOwnership } from "@/lib/api";
const mockFetchOwnership = vi.mocked(fetchOwnership);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useOwnership", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches ownership for given slug", async () => {
    // Given
    const response = {
      data: [{ repo_name: "api", owner_count: 5, team_count: 2, teams: ["@org/backend"] }],
      total: 1,
      page: 1,
      per_page: 50,
    };
    mockFetchOwnership.mockResolvedValue(response);

    // When
    const { result } = renderHook(() => useOwnership("my-org"), {
      wrapper: createWrapper(),
    });

    // Then
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(response);
    expect(mockFetchOwnership).toHaveBeenCalledWith("my-org", 1, 50);
  });

  it("starts in loading state", () => {
    // Given
    mockFetchOwnership.mockReturnValue(new Promise(() => {}));

    // When
    const { result } = renderHook(() => useOwnership("my-org"), {
      wrapper: createWrapper(),
    });

    // Then
    expect(result.current.isPending).toBe(true);
  });

  it("returns isError on failure", async () => {
    // Given
    mockFetchOwnership.mockRejectedValue(new Error("network error"));

    // When
    const { result } = renderHook(() => useOwnership("my-org"), {
      wrapper: createWrapper(),
    });

    // Then
    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it("does not fetch when slug is empty", () => {
    // Given / When
    const { result } = renderHook(() => useOwnership(""), {
      wrapper: createWrapper(),
    });

    // Then
    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchOwnership).not.toHaveBeenCalled();
  });

  it("returns empty data array on empty response", async () => {
    // Given
    const response = { data: [], total: 0, page: 1, per_page: 50 };
    mockFetchOwnership.mockResolvedValue(response);

    // When
    const { result } = renderHook(() => useOwnership("my-org"), {
      wrapper: createWrapper(),
    });

    // Then
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.data).toEqual([]);
  });
});
