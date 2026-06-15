import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchOrgs,
  fetchRepos,
  fetchRepoDetail,
  fetchRepoDeps,
  connectInstallation,
  fetchDependencies,
  fetchDependencyDetail,
  fetchOwnership,
  fetchOwnershipDetail,
} from "@/lib/api";

vi.mock("@/lib/auth", () => ({
  apiFetch: vi.fn(),
}));

import { apiFetch } from "@/lib/auth";
const mockApiFetch = vi.mocked(apiFetch);

beforeEach(() => {
  vi.clearAllMocks();
});

describe("fetchOrgs", () => {
  it("returns organizations on success", async () => {
    const orgs = [{ id: "1", name: "org-1", slug: "org-1" }];
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(orgs),
    } as Response);

    const result = await fetchOrgs();
    expect(result).toEqual(orgs);
    expect(mockApiFetch).toHaveBeenCalledWith("/api/v1/orgs");
  });

  it("throws on failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(fetchOrgs()).rejects.toThrow("Failed to fetch organizations");
  });
});

describe("fetchRepos", () => {
  it("returns repos for an org", async () => {
    const response = { data: [{ id: "r1", name: "repo-1" }], total: 1, page: 1, per_page: 25 };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(response),
    } as Response);

    const result = await fetchRepos("org-uuid-123");
    expect(result).toEqual(response);
    expect(mockApiFetch).toHaveBeenCalledWith(
      "/api/v1/orgs/org-uuid-123/repos?page=1&per_page=25"
    );
  });

  it("passes q param when provided", async () => {
    const response = { data: [], total: 0, page: 1, per_page: 25 };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(response),
    } as Response);

    await fetchRepos("my-org", 1, 25, "react");
    expect(mockApiFetch).toHaveBeenCalledWith(
      "/api/v1/orgs/my-org/repos?page=1&per_page=25&q=react"
    );
  });

  it("throws on failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 404 } as Response);
    await expect(fetchRepos("bad-id")).rejects.toThrow("Failed to fetch repositories");
  });
});

describe("connectInstallation", () => {
  it("sends installation_id and returns org", async () => {
    const org = { id: "1", name: "connected-org" };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(org),
    } as Response);

    const result = await connectInstallation(12345);
    expect(result).toEqual(org);
    expect(mockApiFetch).toHaveBeenCalledWith("/api/v1/orgs/connect", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ installation_id: 12345 }),
    });
  });

  it("throws on failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 409 } as Response);
    await expect(connectInstallation(99)).rejects.toThrow("Failed to connect installation");
  });
});

describe("fetchRepoDetail", () => {
  it("returns repo on success", async () => {
    const repo = { id: "r1", name: "atlas" };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(repo),
    } as Response);

    const result = await fetchRepoDetail("my-org", "atlas");
    expect(result).toEqual(repo);
    expect(mockApiFetch).toHaveBeenCalledWith("/api/v1/orgs/my-org/repos/atlas");
  });

  it("throws on failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 404 } as Response);
    await expect(fetchRepoDetail("my-org", "bad")).rejects.toThrow("Failed to fetch repository");
  });
});

describe("fetchRepoDeps", () => {
  it("returns dependencies on success", async () => {
    const response = { repo: "atlas", dependencies: [] };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(response),
    } as Response);

    const result = await fetchRepoDeps("my-org", "atlas");
    expect(result).toEqual(response);
    expect(mockApiFetch).toHaveBeenCalledWith(
      "/api/v1/orgs/my-org/repos/atlas/dependencies"
    );
  });

  it("throws on failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(fetchRepoDeps("my-org", "atlas")).rejects.toThrow(
      "Failed to fetch repository dependencies"
    );
  });
});

describe("fetchDependencies", () => {
  it("returns paginated dependencies", async () => {
    const response = { data: [], total: 0, page: 1, per_page: 50 };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(response),
    } as Response);

    const result = await fetchDependencies("my-org");
    expect(result).toEqual(response);
    expect(mockApiFetch).toHaveBeenCalledWith(
      "/api/v1/orgs/my-org/dependencies?page=1&per_page=50"
    );
  });

  it("throws on failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(fetchDependencies("my-org")).rejects.toThrow("Failed to fetch dependencies");
  });
});

describe("fetchDependencyDetail", () => {
  it("returns repos using the dependency", async () => {
    const response = { repos: [{ repo_name: "atlas", version: "1.0" }] };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(response),
    } as Response);

    const result = await fetchDependencyDetail("my-org", "npm", "react");
    expect(result).toEqual(response.repos);
    expect(mockApiFetch).toHaveBeenCalledWith(
      "/api/v1/orgs/my-org/dependencies/npm/react"
    );
  });

  it("returns empty array on 404", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 404 } as Response);
    const result = await fetchDependencyDetail("my-org", "npm", "missing");
    expect(result).toEqual([]);
  });

  it("throws on non-404 failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(fetchDependencyDetail("my-org", "npm", "bad")).rejects.toThrow(
      "Failed to fetch dependency detail"
    );
  });
});

describe("fetchOwnership", () => {
  it("returns paginated ownership", async () => {
    const response = { data: [], total: 0, page: 1, per_page: 50 };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(response),
    } as Response);

    const result = await fetchOwnership("my-org");
    expect(result).toEqual(response);
    expect(mockApiFetch).toHaveBeenCalledWith(
      "/api/v1/orgs/my-org/ownership?page=1&per_page=50"
    );
  });

  it("throws on failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(fetchOwnership("my-org")).rejects.toThrow("Failed to fetch ownership");
  });
});

describe("fetchOwnershipDetail", () => {
  it("returns ownership rules for a repo", async () => {
    const response = { repo: "atlas", rules: [] };
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(response),
    } as Response);

    const result = await fetchOwnershipDetail("my-org", "atlas");
    expect(result).toEqual(response);
    expect(mockApiFetch).toHaveBeenCalledWith("/api/v1/orgs/my-org/ownership/atlas");
  });

  it("throws on failure", async () => {
    mockApiFetch.mockResolvedValue({ ok: false, status: 500 } as Response);
    await expect(fetchOwnershipDetail("my-org", "atlas")).rejects.toThrow(
      "Failed to fetch ownership detail"
    );
  });
});
