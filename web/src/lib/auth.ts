export interface User {
  id: string;
  github_id: number;
  login: string;
  name?: string;
  email?: string;
  avatar_url?: string;
}

let accessToken: string | null = null;

export function setTokens(access: string, refresh: string) {
  accessToken = access;
  localStorage.setItem("atlas_refresh_token", refresh);
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function clearAuth() {
  accessToken = null;
  localStorage.removeItem("atlas_refresh_token");
}

export function hasRefreshToken(): boolean {
  return localStorage.getItem("atlas_refresh_token") !== null;
}

export async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = localStorage.getItem("atlas_refresh_token");
  if (!refreshToken) return false;

  try {
    const res = await fetch("/api/v1/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!res.ok) {
      clearAuth();
      return false;
    }
    const data = await res.json();
    accessToken = data.access_token;
    localStorage.setItem("atlas_refresh_token", data.refresh_token);
    return true;
  } catch {
    clearAuth();
    return false;
  }
}

export async function apiFetch(
  path: string,
  init?: RequestInit,
): Promise<Response> {
  const headers = new Headers(init?.headers);
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  let res = await fetch(path, { ...init, headers });

  if (res.status === 401 && accessToken) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      headers.set("Authorization", `Bearer ${accessToken}`);
      res = await fetch(path, { ...init, headers });
    }
  }

  return res;
}

export async function fetchCurrentUser(): Promise<User | null> {
  try {
    const res = await apiFetch("/api/v1/auth/me");
    if (!res.ok) return null;
    return res.json();
  } catch {
    return null;
  }
}

export function extractTokensFromHash(): {
  access: string;
  refresh: string;
} | null {
  const hash = window.location.hash.substring(1);
  if (!hash) return null;

  const params = new URLSearchParams(hash);
  const access = params.get("access_token");
  const refresh = params.get("refresh_token");

  if (access && refresh) {
    window.history.replaceState(null, "", window.location.pathname);
    return { access, refresh };
  }
  return null;
}
