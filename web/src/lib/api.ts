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

export async function fetchRepos(slug: string): Promise<Repository[]> {
  const res = await apiFetch(`/api/v1/orgs/${slug}/repos`);
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
): Promise<DependencyListResponse> {
  const res = await apiFetch(
    `/api/v1/orgs/${slug}/dependencies?page=${page}&per_page=${perPage}`,
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
