import { apiFetch } from "@/lib/auth";

export interface DependencyWithCount {
  ecosystem: string;
  name: string;
  repo_count: number;
}

export interface DepDetail {
  repo_name: string;
  repo_slug: string;
  version: string;
  dep_type: string;
  source_file: string;
}

export interface DependencyListResponse {
  data: DependencyWithCount[];
  total: number;
  page: number;
  per_page: number;
}

export interface Organization {
  id: string;
  github_id: number;
  name: string;
  slug: string;
  github_installation_id?: number;
  owner_id: string;
  last_synced_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Repository {
  id: string;
  org_id: string;
  github_id: number;
  name: string;
  full_name: string;
  description?: string;
  default_branch: string;
  language?: string;
  private: boolean;
  fork: boolean;
  stars: number;
  last_synced_at?: string;
  created_at: string;
  updated_at: string;
}

export async function fetchOrgs(): Promise<Organization[]> {
  const res = await apiFetch("/api/v1/orgs");
  if (!res.ok) throw new Error("Failed to fetch organizations");
  return res.json();
}

export interface RepoListResponse {
  data: Repository[];
  total: number;
  page: number;
  per_page: number;
}

export async function fetchRepos(
  slug: string,
  page = 1,
  perPage = 25,
  q = "",
): Promise<RepoListResponse> {
  const params = new URLSearchParams({
    page: String(page),
    per_page: String(perPage),
  });
  if (q) params.set("q", q);
  const res = await apiFetch(`/api/v1/orgs/${slug}/repos?${params}`);
  if (!res.ok) throw new Error("Failed to fetch repositories");
  return res.json();
}

export async function connectInstallation(installationID: number): Promise<Organization> {
  const res = await apiFetch("/api/v1/orgs/connect", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ installation_id: installationID }),
  });
  if (!res.ok) throw new Error("Failed to connect installation");
  return res.json();
}

export async function fetchDependencies(
  slug: string,
  page = 1,
  perPage = 50,
  q = "",
): Promise<DependencyListResponse> {
  const params = new URLSearchParams({
    page: String(page),
    per_page: String(perPage),
  });
  if (q) params.set("q", q);
  const res = await apiFetch(
    `/api/v1/orgs/${slug}/dependencies?${params}`,
  );
  if (!res.ok) throw new Error("Failed to fetch dependencies");
  return res.json();
}

export async function fetchDependencyDetail(
  slug: string,
  ecosystem: string,
  name: string,
): Promise<DepDetail[]> {
  const res = await apiFetch(
    `/api/v1/orgs/${slug}/dependencies/${ecosystem}/${name}`,
  );
  if (res.status === 404) return [];
  if (!res.ok) throw new Error("Failed to fetch dependency detail");
  const data = await res.json();
  return data.repos ?? [];
}

// --- Ownership types ---

export interface RepoOwnerSummary {
  repo_name: string;
  owner_count: number;
  team_count: number;
  teams: string[];
}

export interface OwnerRule {
  pattern: string;
  owner: string;
  owner_type: string;
  line_number?: number;
}

export interface OwnershipListResponse {
  data: RepoOwnerSummary[];
  total: number;
  page: number;
  per_page: number;
}

export interface OwnershipDetailResponse {
  repo: string;
  rules: OwnerRule[];
}

// --- Ownership fetch functions ---

export async function fetchOwnership(
  slug: string,
  page = 1,
  perPage = 50,
): Promise<OwnershipListResponse> {
  const res = await apiFetch(
    `/api/v1/orgs/${slug}/ownership?page=${page}&per_page=${perPage}`,
  );
  if (!res.ok) throw new Error("Failed to fetch ownership");
  return res.json();
}

export async function fetchOwnershipDetail(
  slug: string,
  repo: string,
): Promise<OwnershipDetailResponse> {
  const res = await apiFetch(`/api/v1/orgs/${slug}/ownership/${repo}`);
  if (!res.ok) throw new Error("Failed to fetch ownership detail");
  return res.json();
}
