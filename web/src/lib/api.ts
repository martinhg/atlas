import { apiFetch } from "@/lib/auth";

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

export async function fetchRepos(orgID: string): Promise<Repository[]> {
  const res = await apiFetch(`/api/v1/orgs/${orgID}/repos`);
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
