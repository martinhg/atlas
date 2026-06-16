import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClientProvider, QueryClient } from "@tanstack/react-query";
import { ImpactAnalysisPage } from "@/features/impact/ImpactAnalysisPage";

vi.mock("@/lib/auth", () => ({
  clearAuth: vi.fn(),
}));

vi.mock("@/features/impact/useImpactAnalysis", () => ({
  useImpactAnalysis: vi.fn(),
}));

import { useImpactAnalysis } from "@/features/impact/useImpactAnalysis";
const mockUseImpactAnalysis = vi.mocked(useImpactAnalysis);

const onLogout = vi.fn();
const mutate = vi.fn();

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

function renderPage(slug = "test-org", searchParams = "") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orgs/${slug}/impact${searchParams}`]}>
        <Routes>
          <Route path="/orgs/:slug/impact" element={<ImpactAnalysisPage onLogout={onLogout} />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ImpactAnalysisPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseImpactAnalysis.mockReturnValue({
      mutate,
      data: undefined,
      isPending: false,
      isError: false,
      isIdle: true,
    } as unknown as ReturnType<typeof useImpactAnalysis>);
  });

  it("renders the dependency name input and ecosystem select", () => {
    renderPage();
    expect(screen.getByLabelText(/dependency name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/ecosystem/i)).toBeInTheDocument();
  });

  it("submits the form and calls mutate with dependency and ecosystem", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText(/dependency name/i), "lodash");
    await user.click(screen.getByRole("button", { name: /analyze/i }));

    expect(mutate).toHaveBeenCalledWith({ dependency: "lodash", ecosystem: "npm" });
  });

  it("pre-fills the dependency name from query params", () => {
    renderPage("test-org", "?dependency=lodash&ecosystem=npm");
    expect(screen.getByLabelText(/dependency name/i)).toHaveValue("lodash");
  });

  it("shows loading state while pending", () => {
    mockUseImpactAnalysis.mockReturnValue({
      mutate,
      data: undefined,
      isPending: true,
      isError: false,
      isIdle: false,
    } as unknown as ReturnType<typeof useImpactAnalysis>);

    renderPage();
    expect(screen.getByText(/analyzing/i)).toBeInTheDocument();
  });

  it("shows error state on failure", () => {
    mockUseImpactAnalysis.mockReturnValue({
      mutate,
      data: undefined,
      isPending: false,
      isError: true,
      isIdle: false,
    } as unknown as ReturnType<typeof useImpactAnalysis>);

    renderPage();
    expect(screen.getByText(/failed to analyze impact/i)).toBeInTheDocument();
  });

  it("renders results with risk badge and counts when data is available", () => {
    mockUseImpactAnalysis.mockReturnValue({
      mutate,
      data: mockResponse,
      isPending: false,
      isError: false,
      isIdle: false,
    } as unknown as ReturnType<typeof useImpactAnalysis>);

    renderPage();
    expect(screen.getByText("high")).toBeInTheDocument();
    expect(screen.getByText("org/repo-name")).toBeInTheDocument();
    expect(screen.getByText(/1 repositor/i)).toBeInTheDocument();
    expect(screen.getByText(/1 team/i)).toBeInTheDocument();
  });
});
