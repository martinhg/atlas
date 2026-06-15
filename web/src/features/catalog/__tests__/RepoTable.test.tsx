import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { RepoTable } from "@/features/catalog/RepoTable";
import type { Repository } from "@/lib/api";

const makeRepo = (overrides: Partial<Repository> = {}): Repository => ({
  id: "r1",
  org_id: "o1",
  github_id: 1,
  name: "test-repo",
  full_name: "org/test-repo",
  default_branch: "main",
  private: false,
  fork: false,
  stars: 0,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  ...overrides,
});

function renderTable(repos: Repository[], slug = "test-org") {
  return render(
    <MemoryRouter>
      <RepoTable repos={repos} slug={slug} />
    </MemoryRouter>,
  );
}

describe("RepoTable", () => {
  it("shows empty state when no repos", () => {
    renderTable([]);
    expect(screen.getByText(/no repositories found/i)).toBeInTheDocument();
  });

  it("renders table headers", () => {
    renderTable([makeRepo()]);
    expect(screen.getByText("Repository")).toBeInTheDocument();
    expect(screen.getByText("Language")).toBeInTheDocument();
    expect(screen.getByText("Branch")).toBeInTheDocument();
    expect(screen.getByText("Stars")).toBeInTheDocument();
  });

  it("renders repo name as link to detail page", () => {
    renderTable([makeRepo({ name: "atlas" })], "my-org");
    const link = screen.getByRole("link", { name: "atlas" });
    expect(link).toHaveAttribute("href", "/orgs/my-org/repos/atlas");
  });

  it("renders description when present", () => {
    renderTable([makeRepo({ description: "A cool project" })]);
    expect(screen.getByText("A cool project")).toBeInTheDocument();
  });

  it("renders language when present", () => {
    renderTable([makeRepo({ language: "Go" })]);
    expect(screen.getByText("Go")).toBeInTheDocument();
  });

  it("renders dash when no language", () => {
    renderTable([makeRepo({ language: undefined })]);
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("renders star count", () => {
    renderTable([makeRepo({ stars: 42 })]);
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("renders multiple repos", () => {
    const repos = [
      makeRepo({ id: "1", name: "repo-a" }),
      makeRepo({ id: "2", name: "repo-b" }),
      makeRepo({ id: "3", name: "repo-c" }),
    ];
    renderTable(repos);
    expect(screen.getByText("repo-a")).toBeInTheDocument();
    expect(screen.getByText("repo-b")).toBeInTheDocument();
    expect(screen.getByText("repo-c")).toBeInTheDocument();
  });
});
