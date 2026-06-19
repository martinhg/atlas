import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { DependencyVulnerabilities } from "@/features/dependencies/DependencyVulnerabilities";
import type { VulnerabilityListItem } from "@/lib/api";

vi.mock("@/features/vulnerabilities/useVulnerabilities", () => ({
  useVulnerabilities: vi.fn(),
}));

import { useVulnerabilities } from "@/features/vulnerabilities/useVulnerabilities";
const mockUse = vi.mocked(useVulnerabilities);

const vuln: VulnerabilityListItem = {
  id: "v1",
  osv_id: "GHSA-aaaa",
  cve_id: "CVE-2021-1234",
  ecosystem: "npm",
  package_name: "lodash",
  severity: "critical",
  cvss_score: 9.8,
  affected_repo_count: 1,
  affected_team_count: 1,
};

function renderSection() {
  return render(
    <MemoryRouter>
      <DependencyVulnerabilities slug="test-org" name="lodash" />
    </MemoryRouter>,
  );
}

describe("DependencyVulnerabilities", () => {
  beforeEach(() => vi.clearAllMocks());

  const mockReturn = (over: Record<string, unknown>) =>
    mockUse.mockReturnValue(over as unknown as ReturnType<typeof useVulnerabilities>);

  it("renders the section heading", () => {
    mockReturn({ data: { data: [], total: 0, page: 1, per_page: 100 }, isPending: false, isError: false });
    renderSection();
    expect(screen.getByText("Known Vulnerabilities")).toBeInTheDocument();
  });

  it("fetches vulnerabilities filtered by the package name", () => {
    mockReturn({ data: { data: [], total: 0, page: 1, per_page: 100 }, isPending: false, isError: false });
    renderSection();
    expect(mockUse).toHaveBeenCalledWith("test-org", 1, 100, "", "lodash");
  });

  it("shows the empty state when there are no vulnerabilities", () => {
    mockReturn({ data: { data: [], total: 0, page: 1, per_page: 100 }, isPending: false, isError: false });
    renderSection();
    expect(screen.getByText(/no known vulnerabilities/i)).toBeInTheDocument();
  });

  it("renders a vuln row with a severity badge and a link to its detail page", () => {
    mockReturn({ data: { data: [vuln], total: 1, page: 1, per_page: 100 }, isPending: false, isError: false });
    renderSection();
    expect(screen.getByText("critical")).toBeInTheDocument();
    const link = screen.getByRole("link", { name: "GHSA-aaaa" });
    expect(link).toHaveAttribute("href", "/orgs/test-org/vulnerabilities/v1");
  });

  it("shows the error state on failure", () => {
    mockReturn({ data: undefined, isPending: false, isError: true });
    renderSection();
    expect(screen.getByText(/failed to load vulnerabilities/i)).toBeInTheDocument();
  });
});
