import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useDependencies } from "@/features/dependencies/useDependencies";

vi.mock("@/lib/api", () => ({
  fetchDependencies: vi.fn(),
}));

import { fetchDependencies } from "@/lib/api";
const mockFetchDependencies = vi.mocked(fetchDependencies);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useDependencies", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches dependencies for given slug", async () => {
    const response = {
      data: [{ ecosystem: "npm", name: "react", repo_count: 5 }],
      total: 1,
      page: 1,
      per_page: 50,
    };
    mockFetchDependencies.mockResolvedValue(response);

    const { result } = renderHook(() => useDependencies("my-org"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(response);
    expect(mockFetchDependencies).toHaveBeenCalledWith("my-org", 1, 50);
  });

  it("starts in loading state", () => {
    mockFetchDependencies.mockReturnValue(new Promise(() => {}));

    const { result } = renderHook(() => useDependencies("my-org"), {
      wrapper: createWrapper(),
    });

    expect(result.current.isPending).toBe(true);
  });

  it("returns empty data array on empty response", async () => {
    const response = { data: [], total: 0, page: 1, per_page: 50 };
    mockFetchDependencies.mockResolvedValue(response);

    const { result } = renderHook(() => useDependencies("my-org"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.data).toEqual([]);
  });

  it("returns isError on failure", async () => {
    mockFetchDependencies.mockRejectedValue(new Error("network error"));

    const { result } = renderHook(() => useDependencies("my-org"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it("does not fetch when slug is empty", () => {
    const { result } = renderHook(() => useDependencies(""), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchDependencies).not.toHaveBeenCalled();
  });
});
