import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ImpactResultTable } from "@/features/impact/ImpactResultTable";
import type { ImpactAffectedRepo } from "@/lib/api";

const makeRepo = (overrides: Partial<ImpactAffectedRepo> = {}): ImpactAffectedRepo => ({
  id: "repo-1",
  name: "repo-name",
  full_name: "org/repo-name",
  version: "4.17.21",
  dep_type: "direct",
  teams: ["@org/team-frontend", "@user"],
  ...overrides,
});

describe("ImpactResultTable", () => {
  it("shows empty state when no affected repos", () => {
    render(<ImpactResultTable repos={[]} />);
    expect(screen.getByText(/no repositories are affected/i)).toBeInTheDocument();
  });

  it("renders column headers", () => {
    render(<ImpactResultTable repos={[makeRepo()]} />);
    expect(screen.getByText("Repository")).toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
    expect(screen.getByText("Owners")).toBeInTheDocument();
  });

  it("renders repo full name, version, dep type, and owners", () => {
    render(<ImpactResultTable repos={[makeRepo()]} />);
    expect(screen.getByText("org/repo-name")).toBeInTheDocument();
    expect(screen.getByText("4.17.21")).toBeInTheDocument();
    expect(screen.getByText("direct")).toBeInTheDocument();
    expect(screen.getByText("@org/team-frontend, @user")).toBeInTheDocument();
  });

  it("renders multiple rows", () => {
    const repos = [
      makeRepo({ id: "repo-1", full_name: "org/repo-a" }),
      makeRepo({ id: "repo-2", full_name: "org/repo-b" }),
    ];
    render(<ImpactResultTable repos={repos} />);
    expect(screen.getByText("org/repo-a")).toBeInTheDocument();
    expect(screen.getByText("org/repo-b")).toBeInTheDocument();
  });

  it("renders an em dash when a repo has no owners", () => {
    render(<ImpactResultTable repos={[makeRepo({ teams: [] })]} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});
