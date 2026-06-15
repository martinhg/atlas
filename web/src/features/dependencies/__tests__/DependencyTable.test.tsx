import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DependencyTable } from "@/features/dependencies/DependencyTable";
import type { DependencyWithCount } from "@/lib/api";

const makeDep = (overrides: Partial<DependencyWithCount> = {}): DependencyWithCount => ({
  ecosystem: "npm",
  name: "react",
  repo_count: 3,
  ...overrides,
});

function renderTable(deps: DependencyWithCount[], onRowClick?: (dep: DependencyWithCount) => void) {
  return render(
    <DependencyTable deps={deps} onRowClick={onRowClick} />,
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
    expect(screen.getByText("Repos")).toBeInTheDocument();
  });

  it("renders dep name, ecosystem and repo count", () => {
    renderTable([makeDep({ name: "lodash", ecosystem: "npm", repo_count: 7 })]);
    expect(screen.getByText("lodash")).toBeInTheDocument();
    expect(screen.getByText("npm")).toBeInTheDocument();
    expect(screen.getByText("7")).toBeInTheDocument();
  });

  it("renders multiple rows", () => {
    const deps = [
      makeDep({ name: "react", ecosystem: "npm" }),
      makeDep({ name: "express", ecosystem: "npm" }),
    ];
    renderTable(deps);
    expect(screen.getByText("react")).toBeInTheDocument();
    expect(screen.getByText("express")).toBeInTheDocument();
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
