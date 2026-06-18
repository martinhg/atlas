import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { DependencyTable } from "@/features/dependencies/DependencyTable";
import type { DependencyWithCount } from "@/lib/api";

const makeDep = (overrides: Partial<DependencyWithCount> = {}): DependencyWithCount => ({
  ecosystem: "npm",
  name: "react",
  repo_count: 3,
  vuln_count: 0,
  max_severity: "",
  ...overrides,
});

function renderTable(deps: DependencyWithCount[], onRowClick?: (dep: DependencyWithCount) => void) {
  return render(
    <MemoryRouter initialEntries={["/orgs/test-org/dependencies"]}>
      <Routes>
        <Route
          path="/orgs/:slug/dependencies"
          element={<DependencyTable deps={deps} onRowClick={onRowClick} />}
        />
        <Route path="/orgs/:slug/vulnerabilities" element={<div>Vuln dashboard</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("DependencyTable", () => {
  it("shows empty state when no deps", () => {
    renderTable([]);
    expect(screen.getByText(/no dependencies found/i)).toBeInTheDocument();
  });

  it("renders column headers", () => {
    renderTable([makeDep()]);
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Ecosystem")).toBeInTheDocument();
    expect(screen.getByText("Vulnerabilities")).toBeInTheDocument();
    expect(screen.getByText("Repos")).toBeInTheDocument();
  });

  it("renders dep name, ecosystem and repo count", () => {
    renderTable([makeDep({ name: "lodash", ecosystem: "npm", repo_count: 7 })]);
    expect(screen.getByText("lodash")).toBeInTheDocument();
    expect(screen.getByText("npm")).toBeInTheDocument();
    expect(screen.getByText("7")).toBeInTheDocument();
  });

  it("shows a 0 with no badge when the dependency has no vulnerabilities", () => {
    renderTable([makeDep({ vuln_count: 0, max_severity: "" })]);
    expect(screen.getByText("0")).toBeInTheDocument();
    expect(screen.queryByText("critical")).not.toBeInTheDocument();
  });

  it("shows the vuln count and highest-severity badge", () => {
    renderTable([makeDep({ name: "lodash", repo_count: 8, vuln_count: 3, max_severity: "critical" })]);
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("critical")).toBeInTheDocument();
  });

  it("navigates to the filtered vuln dashboard when the count is clicked", async () => {
    const user = userEvent.setup();
    const onRowClick = vi.fn();
    renderTable([makeDep({ name: "lodash", vuln_count: 2, max_severity: "high" })], onRowClick);

    await user.click(screen.getByRole("button", { name: /view 2 vulnerabilities for lodash/i }));

    // The vuln count click must NOT trigger the row click.
    expect(onRowClick).not.toHaveBeenCalled();
    expect(screen.getByText("Vuln dashboard")).toBeInTheDocument();
  });

  it("calls onRowClick when a row is clicked", async () => {
    const user = userEvent.setup();
    const onRowClick = vi.fn();
    const dep = makeDep({ name: "vue", ecosystem: "npm", repo_count: 2 });

    renderTable([dep], onRowClick);

    await user.click(screen.getByText("vue"));
    expect(onRowClick).toHaveBeenCalledWith(dep);
  });
});
