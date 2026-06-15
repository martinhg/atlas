import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { DependencyListPage } from "@/features/dependencies/DependencyListPage";

vi.mock("@/lib/auth", () => ({
  clearAuth: vi.fn(),
}));

const mockDeps = [
  { ecosystem: "npm", name: "react", repo_count: 5 },
  { ecosystem: "npm", name: "lodash", repo_count: 3 },
];

vi.mock("@/features/dependencies/useDependencies", () => ({
  useDependencies: vi.fn(),
}));

import { useDependencies } from "@/features/dependencies/useDependencies";
const mockUseDependencies = vi.mocked(useDependencies);

const onLogout = vi.fn();

function renderPage(slug = "test-org") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orgs/${slug}/dependencies`]}>
        <Routes>
          <Route
            path="/orgs/:slug/dependencies"
            element={<DependencyListPage onLogout={onLogout} />}
          />
          <Route path="/dashboard" element={<div>Dashboard</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("DependencyListPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state", () => {
    mockUseDependencies.mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    expect(screen.getByText(/loading dependencies/i)).toBeInTheDocument();
  });

  it("shows error state", () => {
    mockUseDependencies.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    expect(screen.getByText(/failed to load dependencies/i)).toBeInTheDocument();
  });

  it("renders dependency rows when loaded", () => {
    mockUseDependencies.mockReturnValue({
      data: { data: mockDeps, total: 2, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    expect(screen.getByText("react")).toBeInTheDocument();
    expect(screen.getByText("lodash")).toBeInTheDocument();
  });

  it("shows empty state message when no dependencies", () => {
    mockUseDependencies.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    expect(screen.getByText(/no dependencies found/i)).toBeInTheDocument();
  });

  it("renders Dependencies heading", () => {
    mockUseDependencies.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    expect(screen.getByRole("heading", { name: /dependencies/i })).toBeInTheDocument();
  });

  it("renders Atlas link back to dashboard", () => {
    mockUseDependencies.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    expect(screen.getByRole("link", { name: "Atlas" })).toHaveAttribute("href", "/dashboard");
  });

  it("renders pagination buttons when totalPages > 1", () => {
    mockUseDependencies.mockReturnValue({
      data: { data: mockDeps, total: 100, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    expect(screen.getByRole("button", { name: /previous/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /next/i })).toBeInTheDocument();
    expect(screen.getByText("Page 1 of 2")).toBeInTheDocument();
  });

  it("disables Previous button on first page", () => {
    mockUseDependencies.mockReturnValue({
      data: { data: mockDeps, total: 100, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled();
  });

  it("advances page when Next is clicked", async () => {
    const user = userEvent.setup();
    mockUseDependencies.mockReturnValue({
      data: { data: mockDeps, total: 150, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencies>);

    renderPage();
    await user.click(screen.getByRole("button", { name: /next/i }));

    // After clicking Next, useDependencies should be called with page 2
    expect(mockUseDependencies).toHaveBeenCalledWith("test-org", 2, 50);
  });
});
