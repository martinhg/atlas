import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { OwnershipTable } from "@/features/ownership/OwnershipTable";
import type { RepoOwnerSummary } from "@/lib/api";

const makeSummary = (overrides: Partial<RepoOwnerSummary> = {}): RepoOwnerSummary => ({
  repo_name: "api",
  owner_count: 5,
  team_count: 2,
  teams: ["@org/backend", "@org/platform"],
  ...overrides,
});

function renderTable(data: RepoOwnerSummary[], slug = "test-org") {
  return render(
    <MemoryRouter>
      <OwnershipTable data={data} slug={slug} />
    </MemoryRouter>,
  );
}

describe("OwnershipTable", () => {
  it("shows empty state when no data", () => {
    renderTable([]);
    expect(screen.getByText(/no ownership data found/i)).toBeInTheDocument();
  });

  it("renders column headers", () => {
    renderTable([makeSummary()]);
    expect(screen.getByText("Repository")).toBeInTheDocument();
    expect(screen.getByText("Owners")).toBeInTheDocument();
    expect(screen.getByText("Teams")).toBeInTheDocument();
    expect(screen.getByText("Team Names")).toBeInTheDocument();
  });

  it("renders repo name, owner count, team count", () => {
    renderTable([makeSummary({ repo_name: "web", owner_count: 3, team_count: 1 })]);
    expect(screen.getByText("web")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("renders repo name as a link to the detail page", () => {
    renderTable([makeSummary({ repo_name: "atlas-api" })], "my-org");
    const link = screen.getByRole("link", { name: "atlas-api" });
    expect(link).toHaveAttribute("href", "/orgs/my-org/ownership/atlas-api");
  });

  it("renders team names", () => {
    renderTable([makeSummary({ teams: ["@org/backend", "@org/platform"] })]);
    expect(screen.getByText("@org/backend")).toBeInTheDocument();
    expect(screen.getByText("@org/platform")).toBeInTheDocument();
  });

  it("renders multiple rows", () => {
    const data = [
      makeSummary({ repo_name: "api" }),
      makeSummary({ repo_name: "web" }),
    ];
    renderTable(data);
    expect(screen.getByText("api")).toBeInTheDocument();
    expect(screen.getByText("web")).toBeInTheDocument();
  });

  it("shows dash when teams array is empty", () => {
    renderTable([makeSummary({ teams: [], team_count: 0 })]);
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});
