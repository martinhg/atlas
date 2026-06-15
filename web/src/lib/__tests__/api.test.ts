import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchOrgs, fetchRepos, connectInstallation } from "@/lib/api";

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
