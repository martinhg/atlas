import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useOwnershipDetail } from "@/features/ownership/useOwnershipDetail";

vi.mock("@/lib/api", () => ({
  fetchOwnershipDetail: vi.fn(),
}));

import { fetchOwnershipDetail } from "@/lib/api";
const mockFetchOwnershipDetail = vi.mocked(fetchOwnershipDetail);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useOwnershipDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches ownership detail for given slug and repo", async () => {
    // Given
    const response = {
      repo: "api",
      rules: [
        { pattern: "*.go", owner: "@org/backend", owner_type: "team", line_number: 1 },
      ],
    };
    mockFetchOwnershipDetail.mockResolvedValue(response);

    // When
    const { result } = renderHook(() => useOwnershipDetail("my-org", "api"), {
      wrapper: createWrapper(),
    });

    // Then
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(response);
    expect(mockFetchOwnershipDetail).toHaveBeenCalledWith("my-org", "api");
  });

  it("starts in loading state", () => {
    // Given
    mockFetchOwnershipDetail.mockReturnValue(new Promise(() => {}));

    // When
    const { result } = renderHook(() => useOwnershipDetail("my-org", "api"), {
      wrapper: createWrapper(),
    });

    // Then
    expect(result.current.isPending).toBe(true);
  });

  it("returns isError on failure", async () => {
    // Given
    mockFetchOwnershipDetail.mockRejectedValue(new Error("server error"));

    // When
    const { result } = renderHook(() => useOwnershipDetail("my-org", "api"), {
      wrapper: createWrapper(),
    });

    // Then
    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it("does not fetch when slug is empty", () => {
    // Given / When
    const { result } = renderHook(() => useOwnershipDetail("", "api"), {
      wrapper: createWrapper(),
    });

    // Then
    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchOwnershipDetail).not.toHaveBeenCalled();
  });

  it("does not fetch when repo is empty", () => {
    // Given / When
    const { result } = renderHook(() => useOwnershipDetail("my-org", ""), {
      wrapper: createWrapper(),
    });

    // Then
    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchOwnershipDetail).not.toHaveBeenCalled();
  });

  it("returns empty rules array when repo has no ownership", async () => {
    // Given
    const response = { repo: "tools", rules: [] };
    mockFetchOwnershipDetail.mockResolvedValue(response);

    // When
    const { result } = renderHook(() => useOwnershipDetail("my-org", "tools"), {
      wrapper: createWrapper(),
    });

    // Then
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.rules).toEqual([]);
  });
});
