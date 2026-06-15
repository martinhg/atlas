import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useDependencyDetail } from "@/features/dependencies/useDependencyDetail";

vi.mock("@/lib/api", () => ({
  fetchDependencyDetail: vi.fn(),
}));

import { fetchDependencyDetail } from "@/lib/api";
const mockFetchDetail = vi.mocked(fetchDependencyDetail);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useDependencyDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches detail for given slug/ecosystem/name", async () => {
    const detail = [
      { repo_name: "web-app", repo_slug: "web-app", version: "^18.2.0", dep_type: "dep", source_file: "package.json" },
    ];
    mockFetchDetail.mockResolvedValue(detail);

    const { result } = renderHook(
      () => useDependencyDetail("my-org", "npm", "react"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(detail);
    expect(mockFetchDetail).toHaveBeenCalledWith("my-org", "npm", "react");
  });

  it("starts in loading state", () => {
    mockFetchDetail.mockReturnValue(new Promise(() => {}));

    const { result } = renderHook(
      () => useDependencyDetail("my-org", "npm", "react"),
      { wrapper: createWrapper() },
    );

    expect(result.current.isPending).toBe(true);
  });

  it("returns empty array when dependency not found", async () => {
    mockFetchDetail.mockResolvedValue([]);

    const { result } = renderHook(
      () => useDependencyDetail("my-org", "npm", "unknown-pkg"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([]);
  });

  it("returns isError on failure", async () => {
    mockFetchDetail.mockRejectedValue(new Error("server error"));

    const { result } = renderHook(
      () => useDependencyDetail("my-org", "npm", "react"),
      { wrapper: createWrapper() },
    );

    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it("does not fetch when slug is empty", () => {
    const { result } = renderHook(
      () => useDependencyDetail("", "npm", "react"),
      { wrapper: createWrapper() },
    );

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchDetail).not.toHaveBeenCalled();
  });
});
