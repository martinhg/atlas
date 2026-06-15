import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

function renderPage(slug = "test-org") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orgs/${slug}/repos`]}>
        <Routes>
          <Route
            path="/orgs/:slug/repos"
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
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByText(/loading repositories/i)).toBeInTheDocument();
  });

  it("shows error state", () => {
    mockUseRepos.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("fail"),
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByText(/failed to load repositories/i)).toBeInTheDocument();
  });

  it("renders repos when loaded", async () => {
    mockUseRepos.mockReturnValue({
      data: { data: mockRepos, total: 1, page: 1, per_page: 25 },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("atlas")).toBeInTheDocument();
    });
    expect(screen.getByText("1 repositories")).toBeInTheDocument();
  });

  it("renders Repositories heading", () => {
    mockUseRepos.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 25 },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByRole("heading", { name: "Repositories" })).toBeInTheDocument();
  });

  it("renders Atlas link back to dashboard", () => {
    mockUseRepos.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 25 },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByRole("link", { name: "Atlas" })).toHaveAttribute(
      "href",
      "/dashboard"
    );
  });

  it("renders sign out button", () => {
    mockUseRepos.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 25 },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByRole("button", { name: /sign out/i })).toBeInTheDocument();
  });

  it("renders search input", () => {
    mockUseRepos.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 25 },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByPlaceholderText(/search repositories/i)).toBeInTheDocument();
  });

  it("debounces search input and passes q to hook", async () => {
    const user = userEvent.setup();
    mockUseRepos.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 25 },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();
    const input = screen.getByPlaceholderText(/search repositories/i);
    await user.type(input, "react");

    await waitFor(() => {
      expect(mockUseRepos).toHaveBeenCalledWith("test-org", 1, 25, "react");
    });
  });

  it("renders pagination when totalPages > 1", () => {
    mockUseRepos.mockReturnValue({
      data: { data: mockRepos, total: 50, page: 1, per_page: 25 },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useRepos>);

    renderPage();
    expect(screen.getByRole("button", { name: /previous/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /next/i })).toBeInTheDocument();
    expect(screen.getByText("Page 1 of 2")).toBeInTheDocument();
  });
});
