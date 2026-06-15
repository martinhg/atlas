import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { DependencyDetailTable } from "@/features/dependencies/DependencyDetailTable";
import type { DepDetail } from "@/lib/api";

const makeDetail = (overrides: Partial<DepDetail> = {}): DepDetail => ({
  repo_name: "web-app",
  repo_slug: "web-app",
  version: "^18.2.0",
  dep_type: "dep",
  source_file: "package.json",
  ...overrides,
});

describe("DependencyDetailTable", () => {
  it("shows empty state when no details", () => {
    render(<DependencyDetailTable repos={[]} />);
    expect(screen.getByText(/not used in any repository/i)).toBeInTheDocument();
  });

  it("renders column headers", () => {
    render(<DependencyDetailTable repos={[makeDetail()]} />);
    expect(screen.getByText("Repository")).toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
    expect(screen.getByText("Source File")).toBeInTheDocument();
  });

  it("renders repo name, version, dep type, and source file", () => {
    render(<DependencyDetailTable repos={[makeDetail({ repo_name: "atlas", version: "^19.0.0", dep_type: "devDep", source_file: "apps/web/package.json" })]} />);
    expect(screen.getByText("atlas")).toBeInTheDocument();
    expect(screen.getByText("^19.0.0")).toBeInTheDocument();
    expect(screen.getByText("devDep")).toBeInTheDocument();
    expect(screen.getByText("apps/web/package.json")).toBeInTheDocument();
  });

  it("renders multiple rows", () => {
    const repos = [
      makeDetail({ repo_name: "repo-a", repo_slug: "repo-a" }),
      makeDetail({ repo_name: "repo-b", repo_slug: "repo-b" }),
    ];
    render(<DependencyDetailTable repos={repos} />);
    expect(screen.getByText("repo-a")).toBeInTheDocument();
    expect(screen.getByText("repo-b")).toBeInTheDocument();
  });
});
