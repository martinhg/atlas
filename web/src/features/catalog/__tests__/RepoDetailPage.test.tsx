import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RepoDetailPage } from "@/features/catalog/RepoDetailPage";

vi.mock("@/lib/auth", () => ({
  clearAuth: vi.fn(),
}));

vi.mock("@/features/catalog/useRepoDetail", () => ({
  useRepoDetail: vi.fn(),
}));

vi.mock("@/features/catalog/useRepoDeps", () => ({
  useRepoDeps: vi.fn(),
}));

vi.mock("@/features/ownership/useOwnershipDetail", () => ({
  useOwnershipDetail: vi.fn(),
}));

import { useRepoDetail } from "@/features/catalog/useRepoDetail";
import { useRepoDeps } from "@/features/catalog/useRepoDeps";
import { useOwnershipDetail } from "@/features/ownership/useOwnershipDetail";

const mockUseRepoDetail = vi.mocked(useRepoDetail);
const mockUseRepoDeps = vi.mocked(useRepoDeps);
const mockUseOwnershipDetail = vi.mocked(useOwnershipDetail);

const onLogout = vi.fn();

function renderPage(slug = "test-org", name = "atlas") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orgs/${slug}/repos/${name}`]}>
        <Routes>
          <Route
            path="/orgs/:slug/repos/:name"
            element={<RepoDetailPage onLogout={onLogout} />}
          />
          <Route path="/dashboard" element={<div>Dashboard</div>} />
          <Route path="/orgs/:slug/repos" element={<div>Repo List</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const mockRepo = {
  id: "r1",
  org_id: "o1",
  github_id: 1,
  name: "atlas",
  full_name: "nesbite/atlas",
  description: "Engineering Intelligence Platform",
  default_branch: "main",
  language: "Go",
  private: false,
  fork: false,
  stars: 42,
  created_at: "2024-01-01",
  updated_at: "2024-01-01",
};

const mockDeps = {
  repo: "atlas",
  dependencies: [
    { ecosystem: "go", name: "chi", version: "v5.0.0", dep_type: "direct", source_file: "go.mod" },
    { ecosystem: "go", name: "pgx", version: "v5.0.0", dep_type: "direct", source_file: "go.mod" },
  ],
};

const mockOwners = {
  repo: "atlas",
  rules: [
    { pattern: "*", owner: "@team-platform", owner_type: "team" },
    { pattern: "/web/*", owner: "@team-frontend", owner_type: "team" },
  ],
};

function setAllLoading() {
  mockUseRepoDetail.mockReturnValue({
    data: undefined, isPending: true, isError: false,
  } as unknown as ReturnType<typeof useRepoDetail>);
  mockUseRepoDeps.mockReturnValue({
    data: undefined, isPending: true, isError: false,
  } as unknown as ReturnType<typeof useRepoDeps>);
  mockUseOwnershipDetail.mockReturnValue({
    data: undefined, isPending: true, isError: false,
  } as unknown as ReturnType<typeof useOwnershipDetail>);
}

function setAllLoaded() {
  mockUseRepoDetail.mockReturnValue({
    data: mockRepo, isPending: false, isError: false,
  } as unknown as ReturnType<typeof useRepoDetail>);
  mockUseRepoDeps.mockReturnValue({
    data: mockDeps, isPending: false, isError: false,
  } as unknown as ReturnType<typeof useRepoDeps>);
  mockUseOwnershipDetail.mockReturnValue({
    data: mockOwners, isPending: false, isError: false,
  } as unknown as ReturnType<typeof useOwnershipDetail>);
}

describe("RepoDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state", () => {
    setAllLoading();
    renderPage();
    expect(screen.getByText(/loading repository details/i)).toBeInTheDocument();
  });

  it("shows error state when repo fetch fails", () => {
    mockUseRepoDetail.mockReturnValue({
      data: undefined, isPending: false, isError: true,
    } as unknown as ReturnType<typeof useRepoDetail>);
    mockUseRepoDeps.mockReturnValue({
      data: undefined, isPending: false, isError: false,
    } as unknown as ReturnType<typeof useRepoDeps>);
    mockUseOwnershipDetail.mockReturnValue({
      data: undefined, isPending: false, isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    renderPage();
    expect(screen.getByText(/failed to load repository/i)).toBeInTheDocument();
  });

  it("renders repo info when loaded", () => {
    setAllLoaded();
    renderPage();
    expect(screen.getByText("nesbite/atlas")).toBeInTheDocument();
    expect(screen.getByText("Engineering Intelligence Platform")).toBeInTheDocument();
    expect(screen.getByText("Go")).toBeInTheDocument();
    expect(screen.getByText("42 stars")).toBeInTheDocument();
  });

  it("renders dependencies section", () => {
    setAllLoaded();
    renderPage();
    expect(screen.getByText("chi")).toBeInTheDocument();
    expect(screen.getByText("pgx")).toBeInTheDocument();
  });

  it("renders ownership section", () => {
    setAllLoaded();
    renderPage();
    expect(screen.getByText("@team-platform")).toBeInTheDocument();
    expect(screen.getByText("@team-frontend")).toBeInTheDocument();
    expect(screen.getByText("/web/*")).toBeInTheDocument();
  });

  it("renders breadcrumb with link to repo list", () => {
    setAllLoaded();
    renderPage();
    const repoListLink = screen.getByRole("link", { name: "Repositories" });
    expect(repoListLink).toHaveAttribute("href", "/orgs/test-org/repos");
  });

  it("renders Atlas link to dashboard", () => {
    setAllLoaded();
    renderPage();
    expect(screen.getByRole("link", { name: "Atlas" })).toHaveAttribute("href", "/dashboard");
  });

  it("shows empty dependencies message when none", () => {
    mockUseRepoDetail.mockReturnValue({
      data: mockRepo, isPending: false, isError: false,
    } as unknown as ReturnType<typeof useRepoDetail>);
    mockUseRepoDeps.mockReturnValue({
      data: { repo: "atlas", dependencies: [] }, isPending: false, isError: false,
    } as unknown as ReturnType<typeof useRepoDeps>);
    mockUseOwnershipDetail.mockReturnValue({
      data: { repo: "atlas", rules: [] }, isPending: false, isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    renderPage();
    expect(screen.getByText(/no dependencies found/i)).toBeInTheDocument();
  });

  it("shows empty ownership message when no rules", () => {
    mockUseRepoDetail.mockReturnValue({
      data: mockRepo, isPending: false, isError: false,
    } as unknown as ReturnType<typeof useRepoDetail>);
    mockUseRepoDeps.mockReturnValue({
      data: { repo: "atlas", dependencies: [] }, isPending: false, isError: false,
    } as unknown as ReturnType<typeof useRepoDeps>);
    mockUseOwnershipDetail.mockReturnValue({
      data: { repo: "atlas", rules: [] }, isPending: false, isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    renderPage();
    expect(screen.getByText(/no codeowners rules found/i)).toBeInTheDocument();
  });
});
