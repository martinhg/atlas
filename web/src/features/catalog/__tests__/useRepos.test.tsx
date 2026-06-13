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

  it("fetches repos for given orgID", async () => {
    const repos: Awaited<ReturnType<typeof fetchRepos>> = [
      { id: "r1", org_id: "o1", github_id: 1, name: "repo-1", full_name: "org/repo-1", default_branch: "main", language: "Go", private: false, fork: false, stars: 0, created_at: "", updated_at: "" },
    ];
    mockFetchRepos.mockResolvedValue(repos);

    const { result } = renderHook(() => useRepos("org-123"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(repos);
    expect(mockFetchRepos).toHaveBeenCalledWith("org-123");
  });

  it("does not fetch when orgID is empty", () => {
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
