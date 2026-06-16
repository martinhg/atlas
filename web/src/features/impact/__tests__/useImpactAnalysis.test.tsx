import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useImpactAnalysis } from "@/features/impact/useImpactAnalysis";

vi.mock("@/lib/api", () => ({
  analyzeImpact: vi.fn(),
}));

import { analyzeImpact } from "@/lib/api";
const mockAnalyzeImpact = vi.mocked(analyzeImpact);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

const mockResponse = {
  dependency: { name: "lodash", ecosystem: "npm" },
  affected_repos: [
    {
      id: "repo-1",
      name: "repo-name",
      full_name: "org/repo-name",
      version: "4.17.21",
      dep_type: "direct",
      teams: ["@org/team-frontend"],
    },
  ],
  version_distribution: [{ version: "4.17.21", count: 1 }],
  risk_score: 7.5,
  risk_level: "high" as const,
  total_repos: 1,
  total_teams: 1,
};

describe("useImpactAnalysis", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("starts idle", () => {
    const { result } = renderHook(() => useImpactAnalysis("my-org"), {
      wrapper: createWrapper(),
    });

    expect(result.current.isIdle).toBe(true);
    expect(mockAnalyzeImpact).not.toHaveBeenCalled();
  });

  it("calls analyzeImpact with slug and body on mutate", async () => {
    mockAnalyzeImpact.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useImpactAnalysis("my-org"), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ dependency: "lodash", ecosystem: "npm" });
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockAnalyzeImpact).toHaveBeenCalledWith("my-org", {
      dependency: "lodash",
      ecosystem: "npm",
    });
    expect(result.current.data).toEqual(mockResponse);
  });

  it("is pending while the request is in flight", async () => {
    mockAnalyzeImpact.mockReturnValue(new Promise(() => {}));

    const { result } = renderHook(() => useImpactAnalysis("my-org"), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ dependency: "lodash", ecosystem: "npm" });
    });

    await waitFor(() => expect(result.current.isPending).toBe(true));
  });

  it("returns isError on failure", async () => {
    mockAnalyzeImpact.mockRejectedValue(new Error("server error"));

    const { result } = renderHook(() => useImpactAnalysis("my-org"), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.mutate({ dependency: "lodash", ecosystem: "npm" });
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
