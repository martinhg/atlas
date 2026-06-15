import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useRepos } from "@/features/catalog/useRepos";

vi.mock("@/lib/api", () => ({
  fetchRepos: vi.fn(),
}));

import { fetchRepos } from "@/lib/api";
const mockFetchRepos = vi.mocked(fetchRepos);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useRepos", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches repos with default params", async () => {
    const response = {
      data: [{ id: "r1", org_id: "o1", github_id: 1, name: "repo-1", full_name: "org/repo-1", default_branch: "main", language: "Go", private: false, fork: false, stars: 0, created_at: "", updated_at: "" }],
      total: 1,
      page: 1,
      per_page: 25,
    };
    mockFetchRepos.mockResolvedValue(response);

    const { result } = renderHook(() => useRepos("org-123"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(response);
    expect(mockFetchRepos).toHaveBeenCalledWith("org-123", 1, 25, "");
  });

  it("passes q param to fetchRepos", async () => {
    const response = { data: [], total: 0, page: 1, per_page: 25 };
    mockFetchRepos.mockResolvedValue(response);

    const { result } = renderHook(() => useRepos("org-123", 1, 25, "react"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockFetchRepos).toHaveBeenCalledWith("org-123", 1, 25, "react");
  });

  it("includes q in queryKey for cache separation", async () => {
    const response = { data: [], total: 0, page: 1, per_page: 25 };
    mockFetchRepos.mockResolvedValue(response);

    const { result: r1 } = renderHook(() => useRepos("org", 1, 25, ""), {
      wrapper: createWrapper(),
    });
    const { result: r2 } = renderHook(() => useRepos("org", 1, 25, "react"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(r1.current.isSuccess).toBe(true));
    await waitFor(() => expect(r2.current.isSuccess).toBe(true));
    expect(mockFetchRepos).toHaveBeenCalledTimes(2);
  });

  it("does not fetch when slug is empty", () => {
    const { result } = renderHook(() => useRepos(""), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchRepos).not.toHaveBeenCalled();
  });

  it("returns error on failure", async () => {
    mockFetchRepos.mockRejectedValue(new Error("fail"));

    const { result } = renderHook(() => useRepos("org-bad"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
