import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { DependencyDetailPage } from "@/features/dependencies/DependencyDetailPage";

vi.mock("@/lib/auth", () => ({
  clearAuth: vi.fn(),
}));

const mockRepos = [
  { repo_name: "web-app", repo_slug: "web-app", version: "^18.2.0", dep_type: "dep", source_file: "package.json" },
];

vi.mock("@/features/dependencies/useDependencyDetail", () => ({
  useDependencyDetail: vi.fn(),
}));

// Stub the vulnerabilities hook used by the DependencyVulnerabilities section so
// this page test stays focused on the dependency detail behavior.
vi.mock("@/features/vulnerabilities/useVulnerabilities", () => ({
  useVulnerabilities: vi.fn(() => ({
    data: { data: [], total: 0, page: 1, per_page: 100 },
    isPending: false,
    isError: false,
  })),
}));

import { useDependencyDetail } from "@/features/dependencies/useDependencyDetail";
const mockUseDetail = vi.mocked(useDependencyDetail);

const onLogout = vi.fn();

function renderPage(slug = "test-org", ecosystem = "npm", name = "react") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/orgs/${slug}/dependencies/${ecosystem}/${name}`]}>
        <Routes>
          <Route
            path="/orgs/:slug/dependencies/:ecosystem/*"
            element={<DependencyDetailPage onLogout={onLogout} />}
          />
          <Route
            path="/orgs/:slug/dependencies"
            element={<div>Dependencies list</div>}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("DependencyDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading skeleton", () => {
    mockUseDetail.mockReturnValue({
      data: undefined,
      isPending: true,
      isError: false,
    } as unknown as ReturnType<typeof useDependencyDetail>);

    renderPage();
    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it("shows error state", () => {
    mockUseDetail.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
    } as unknown as ReturnType<typeof useDependencyDetail>);

    renderPage();
    expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
  });

  it("renders repo rows when loaded", () => {
    mockUseDetail.mockReturnValue({
      data: mockRepos,
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencyDetail>);

    renderPage();
    expect(screen.getByText("web-app")).toBeInTheDocument();
    expect(screen.getByText("^18.2.0")).toBeInTheDocument();
  });

  it("shows not-found message for empty result", () => {
    mockUseDetail.mockReturnValue({
      data: [],
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencyDetail>);

    renderPage();
    expect(screen.getByText(/not used in any repository/i)).toBeInTheDocument();
  });

  it("renders package name as heading", () => {
    mockUseDetail.mockReturnValue({
      data: mockRepos,
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencyDetail>);

    renderPage("test-org", "npm", "react");
    expect(screen.getByRole("heading", { name: /react/i })).toBeInTheDocument();
  });

  it("renders back link to dependencies list", () => {
    mockUseDetail.mockReturnValue({
      data: mockRepos,
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencyDetail>);

    renderPage("test-org", "npm", "react");
    expect(
      screen.getByRole("link", { name: /dependencies/i }),
    ).toHaveAttribute("href", "/orgs/test-org/dependencies");
  });

  it("renders scoped package name correctly via wildcard route", () => {
    mockUseDetail.mockReturnValue({
      data: mockRepos,
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencyDetail>);

    renderPage("test-org", "npm", "@types/react");
    expect(screen.getByRole("heading", { name: "@types/react" })).toBeInTheDocument();
    expect(mockUseDetail).toHaveBeenCalledWith("test-org", "npm", "@types/react");
  });

  it("renders an Analyze Impact link pre-filled with the dependency and ecosystem", () => {
    mockUseDetail.mockReturnValue({
      data: mockRepos,
      isPending: false,
      isError: false,
    } as unknown as ReturnType<typeof useDependencyDetail>);

    renderPage("test-org", "npm", "react");
    expect(
      screen.getByRole("link", { name: /analyze impact/i }),
    ).toHaveAttribute("href", "/orgs/test-org/impact?dependency=react&ecosystem=npm");
  });
});
