import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { OwnershipDetailPage } from "@/features/ownership/OwnershipDetailPage";

vi.mock("@/lib/auth", () => ({
  clearAuth: vi.fn(),
}));

vi.mock("@/features/ownership/useOwnershipDetail", () => ({
  useOwnershipDetail: vi.fn(),
}));

import { useOwnershipDetail } from "@/features/ownership/useOwnershipDetail";
const mockUseOwnershipDetail = vi.mocked(useOwnershipDetail);

const onLogout = vi.fn();

const mockRules = [
  { pattern: "*.go", owner: "@org/backend", owner_type: "team", line_number: 1 },
  { pattern: "*.ts", owner: "@org/frontend", owner_type: "team", line_number: 2 },
  { pattern: "docs/", owner: "writer@example.com", owner_type: "email", line_number: 3 },
  { pattern: "**", owner: "@admin", owner_type: "user", line_number: 4 },
];

function renderPage(slug = "test-org", repo = "api") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orgs/${slug}/ownership/${repo}`]}>
        <Routes>
          <Route
            path="/orgs/:slug/ownership/:repo"
            element={<OwnershipDetailPage onLogout={onLogout} />}
          />
          <Route
            path="/orgs/:slug/ownership"
            element={<div>Ownership list</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("OwnershipDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state", () => {
    // Given
    mockUseOwnershipDetail.mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it("shows error state", () => {
    // Given
    mockUseOwnershipDetail.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    // When
    renderPage();

    // Then
    expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
  });

  it("shows empty state when no rules", () => {
    // Given
    mockUseOwnershipDetail.mockReturnValue({
      data: { repo: "api", rules: [] },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    // When
    renderPage("test-org", "api");

    // Then
    expect(screen.getByText(/no codeowners rules found for this repository/i)).toBeInTheDocument();
  });

  it("renders rule rows when loaded", () => {
    // Given
    mockUseOwnershipDetail.mockReturnValue({
      data: { repo: "api", rules: mockRules },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    // When
    renderPage();

    // Then
    expect(screen.getByText("*.go")).toBeInTheDocument();
    expect(screen.getByText("*.ts")).toBeInTheDocument();
    expect(screen.getByText("@org/backend")).toBeInTheDocument();
  });

  it("renders repo name in breadcrumb and heading", () => {
    // Given
    mockUseOwnershipDetail.mockReturnValue({
      data: { repo: "api", rules: mockRules },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    // When
    renderPage("test-org", "api");

    // Then
    expect(screen.getByRole("heading", { name: "api" })).toBeInTheDocument();
  });

  it("renders back link to ownership list", () => {
    // Given
    mockUseOwnershipDetail.mockReturnValue({
      data: { repo: "api", rules: mockRules },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    // When
    renderPage("test-org", "api");

    // Then
    expect(
      screen.getByRole("link", { name: /ownership/i }),
    ).toHaveAttribute("href", "/orgs/test-org/ownership");
  });

  it("renders Atlas link back to dashboard", () => {
    // Given
    mockUseOwnershipDetail.mockReturnValue({
      data: { repo: "api", rules: mockRules },
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useOwnershipDetail>);

    // When
    renderPage();

    // Then
    expect(screen.getByRole("link", { name: "Atlas" })).toHaveAttribute("href", "/dashboard");
  });
});
