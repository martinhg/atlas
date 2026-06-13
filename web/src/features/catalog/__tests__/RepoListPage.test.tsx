import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RepoListPage } from "@/features/catalog/RepoListPage";

const mockRepos = [
  {
    id: "r1",
    org_id: "o1",
    github_id: 1,
    name: "atlas",
    full_name: "org/atlas",
    default_branch: "main",
    language: "Go",
    private: false,
    fork: false,
    stars: 10,
    created_at: "2024-01-01",
    updated_at: "2024-01-01",
  },
];

vi.mock("@/features/catalog/useRepos", () => ({
  useRepos: vi.fn(),
}));

import { useRepos } from "@/features/catalog/useRepos";
const mockUseRepos = vi.mocked(useRepos);

function renderPage(orgID = "test-org-id") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orgs/${orgID}/repos`]}>
        <Routes>
          <Route
            path="/orgs/:orgID/repos"
            element={<RepoListPage onLogout={() => {}} />}
          />
          <Route path="/dashboard" element={<div>Dashboard</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe("RepoListPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state", () => {
    mockUseRepos.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByText(/loading repositories/i)).toBeInTheDocument();
  });

  it("shows error state", () => {
    mockUseRepos.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("fail"),
    } as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByText(/failed to load repositories/i)).toBeInTheDocument();
  });

  it("renders repos when loaded", async () => {
    mockUseRepos.mockReturnValue({
      data: mockRepos,
      isLoading: false,
      error: null,
    } as ReturnType<typeof useRepos>);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("atlas")).toBeInTheDocument();
    });
    expect(screen.getByText("1 repositories")).toBeInTheDocument();
  });

  it("renders Repositories heading", () => {
    mockUseRepos.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    } as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByRole("heading", { name: "Repositories" })).toBeInTheDocument();
  });

  it("renders Atlas link back to dashboard", () => {
    mockUseRepos.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    } as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByRole("link", { name: "Atlas" })).toHaveAttribute(
      "href",
      "/dashboard"
    );
  });

  it("renders sign out button", () => {
    mockUseRepos.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    } as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByRole("button", { name: /sign out/i })).toBeInTheDocument();
  });
});
