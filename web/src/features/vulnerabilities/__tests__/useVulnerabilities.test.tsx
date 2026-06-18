import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useVulnerabilities } from "@/features/vulnerabilities/useVulnerabilities";

vi.mock("@/lib/api", () => ({
  fetchVulnerabilities: vi.fn(),
}));

import { fetchVulnerabilities } from "@/lib/api";
const mockFetch = vi.mocked(fetchVulnerabilities);

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useVulnerabilities", () => {
  beforeEach(() => vi.clearAllMocks());

  it("fetches with default params", async () => {
    const response = { data: [], total: 0, page: 1, per_page: 20 };
    mockFetch.mockResolvedValue(response);

    const { result } = renderHook(() => useVulnerabilities("my-org"), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockFetch).toHaveBeenCalledWith("my-org", 1, 20, "", "");
  });

  it("passes severity filter through", async () => {
    mockFetch.mockResolvedValue({ data: [], total: 0, page: 1, per_page: 20 });

    const { result } = renderHook(() => useVulnerabilities("my-org", 2, 20, "critical"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockFetch).toHaveBeenCalledWith("my-org", 2, 20, "critical", "");
  });

  it("passes package filter through", async () => {
    mockFetch.mockResolvedValue({ data: [], total: 0, page: 1, per_page: 20 });

    const { result } = renderHook(() => useVulnerabilities("my-org", 1, 20, "", "lodash"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockFetch).toHaveBeenCalledWith("my-org", 1, 20, "", "lodash");
  });

  it("does not fetch when slug is empty", () => {
    const { result } = renderHook(() => useVulnerabilities(""), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("returns isError on failure", async () => {
    mockFetch.mockRejectedValue(new Error("network error"));
    const { result } = renderHook(() => useVulnerabilities("my-org"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
