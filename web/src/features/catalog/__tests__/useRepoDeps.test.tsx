import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useRepoDeps } from "@/features/catalog/useRepoDeps";

vi.mock("@/lib/api", () => ({
  fetchRepoDeps: vi.fn(),
}));

import { fetchRepoDeps } from "@/lib/api";
const mockFetchRepoDeps = vi.mocked(fetchRepoDeps);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useRepoDeps", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches repo dependencies", async () => {
    const response = {
      repo: "atlas",
      dependencies: [{ ecosystem: "go", name: "chi", version: "v5", dep_type: "direct", source_file: "go.mod" }],
    };
    mockFetchRepoDeps.mockResolvedValue(response);

    const { result } = renderHook(() => useRepoDeps("my-org", "atlas"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(response);
    expect(mockFetchRepoDeps).toHaveBeenCalledWith("my-org", "atlas");
  });

  it("does not fetch when slug is empty", () => {
    const { result } = renderHook(() => useRepoDeps("", "atlas"), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchRepoDeps).not.toHaveBeenCalled();
  });

  it("does not fetch when name is empty", () => {
    const { result } = renderHook(() => useRepoDeps("my-org", ""), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockFetchRepoDeps).not.toHaveBeenCalled();
  });

  it("returns error on failure", async () => {
    mockFetchRepoDeps.mockRejectedValue(new Error("fail"));

    const { result } = renderHook(() => useRepoDeps("my-org", "bad"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
