import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useRepoDetail } from "@/features/catalog/useRepoDetail";

vi.mock("@/lib/api", () => ({
  fetchRepoDetail: vi.fn(),
}));

import { fetchRepoDetail } from "@/lib/api";
const mockFetchRepoDetail = vi.mocked(fetchRepoDetail);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useRepoDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches repo detail", async () => {
    const repo = { id: "r1", name: "atlas", full_name: "org/atlas" };
    mockFetchRepoDetail.mockResolvedValue(repo as any);

    const { result } = renderHook(() => useRepoDetail("my-org", "atlas"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(repo);
    expect(mockFetchRepoDetail).toHaveBeenCalledWith("my-org", "atlas");
  });

  it("does not fetch when slug is empty", () => {
    const { result } = renderHook(() => useRepoDetail("", "atlas"), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchRepoDetail).not.toHaveBeenCalled();
  });

  it("does not fetch when name is empty", () => {
    const { result } = renderHook(() => useRepoDetail("my-org", ""), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchRepoDetail).not.toHaveBeenCalled();
  });

  it("returns error on failure", async () => {
    mockFetchRepoDetail.mockRejectedValue(new Error("not found"));

    const { result } = renderHook(() => useRepoDetail("my-org", "bad"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
