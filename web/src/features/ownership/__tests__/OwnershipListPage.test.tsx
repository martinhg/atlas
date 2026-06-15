import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { OwnershipListPage } from "@/features/ownership/OwnershipListPage";

vi.mock("@/lib/auth", () => ({
  clearAuth: vi.fn(),
}));

vi.mock("@/features/ownership/useOwnership", () => ({
  useOwnership: vi.fn(),
}));

import { useOwnership } from "@/features/ownership/useOwnership";
const mockUseOwnership = vi.mocked(useOwnership);

const onLogout = vi.fn();

const mockSummaries = [
  { repo_name: "api", owner_count: 5, team_count: 2, teams: ["@org/backend"] },
  { repo_name: "web", owner_count: 3, team_count: 1, teams: ["@org/frontend"] },
];

function renderPage(slug = "test-org") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orgs/${slug}/ownership`]}>
        <Routes>
          <Route
            path="/orgs/:slug/ownership"
            element={<OwnershipListPage onLogout={onLogout} />}
          />
          <Route path="/dashboard" element={<div>Dashboard</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("OwnershipListPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state", () => {
    // Given
    mockUseOwnership.mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/loading ownership/i)).toBeInTheDocument();
  });

  it("shows error state", () => {
    // Given
    mockUseOwnership.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/failed to load ownership/i)).toBeInTheDocument();
  });

  it("shows empty state message when no ownership data", () => {
    // Given
    mockUseOwnership.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/no ownership data found/i)).toBeInTheDocument();
  });

  it("renders ownership rows when loaded", () => {
    // Given
    mockUseOwnership.mockReturnValue({
      data: { data: mockSummaries, total: 2, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();

    // Then
    expect(screen.getByText("api")).toBeInTheDocument();
    expect(screen.getByText("web")).toBeInTheDocument();
  });

  it("renders Ownership heading", () => {
    // Given
    mockUseOwnership.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();

    // Then
    expect(screen.getByRole("heading", { name: /ownership/i })).toBeInTheDocument();
  });

  it("renders Atlas link back to dashboard", () => {
    // Given
    mockUseOwnership.mockReturnValue({
      data: { data: [], total: 0, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();

    // Then
    expect(screen.getByRole("link", { name: "Atlas" })).toHaveAttribute("href", "/dashboard");
  });

  it("renders pagination when totalPages > 1", () => {
    // Given
    mockUseOwnership.mockReturnValue({
      data: { data: mockSummaries, total: 100, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();

    // Then
    expect(screen.getByRole("button", { name: /previous/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /next/i })).toBeInTheDocument();
    expect(screen.getByText("Page 1 of 2")).toBeInTheDocument();
  });

  it("disables Previous button on first page", () => {
    // Given
    mockUseOwnership.mockReturnValue({
      data: { data: mockSummaries, total: 100, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();

    // Then
    expect(screen.getByRole("button", { name: /previous/i })).toBeDisabled();
  });

  it("advances page when Next is clicked", async () => {
    // Given
    const user = userEvent.setup();
    mockUseOwnership.mockReturnValue({
      data: { data: mockSummaries, total: 150, page: 1, per_page: 50 },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnership>);

    // When
    renderPage();
    await user.click(screen.getByRole("button", { name: /next/i }));

    // Then
    expect(mockUseOwnership).toHaveBeenCalledWith("test-org", 2, 50);
  });
});
