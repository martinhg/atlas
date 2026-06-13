import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useOrgs } from "@/hooks/useOrgs";

vi.mock("@/lib/api", () => ({
  fetchOrgs: vi.fn(),
}));

import { fetchOrgs } from "@/lib/api";
const mockFetchOrgs = vi.mocked(fetchOrgs);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe("useOrgs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns orgs on success", async () => {
    const orgs = [{ id: "1", name: "org-1", slug: "org-1" }];
    mockFetchOrgs.mockResolvedValue(orgs as any);

    const { result } = renderHook(() => useOrgs(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(orgs);
  });

  it("returns error on failure", async () => {
    mockFetchOrgs.mockRejectedValue(new Error("network error"));

    const { result } = renderHook(() => useOrgs(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("network error");
  });
});
