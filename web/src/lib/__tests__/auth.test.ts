import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import {
  setTokens,
  getAccessToken,
  clearAuth,
  hasRefreshToken,
  extractTokensFromHash,
  refreshAccessToken,
  apiFetch,
  fetchCurrentUser,
} from "@/lib/auth"

describe("auth", () => {
  beforeEach(() => {
    clearAuth()
    localStorage.clear()
    window.location.hash = ""
  })

  describe("setTokens", () => {
    it("stores refresh token in localStorage", () => {
      setTokens("access-123", "refresh-456")
      expect(localStorage.getItem("atlas_refresh_token")).toBe("refresh-456")
    })

    it("keeps access token in memory", () => {
      setTokens("access-123", "refresh-456")
      expect(getAccessToken()).toBe("access-123")
    })
  })

  describe("clearAuth", () => {
    it("removes both tokens", () => {
      setTokens("access-123", "refresh-456")
      clearAuth()
      expect(getAccessToken()).toBeNull()
      expect(localStorage.getItem("atlas_refresh_token")).toBeNull()
    })
  })

  describe("hasRefreshToken", () => {
    it("returns false when no token", () => {
      expect(hasRefreshToken()).toBe(false)
    })

    it("returns true when token is set", () => {
      setTokens("access-123", "refresh-456")
      expect(hasRefreshToken()).toBe(true)
    })
  })

  describe("extractTokensFromHash", () => {
    it("returns null when hash is empty", () => {
      expect(extractTokensFromHash()).toBeNull()
    })

    it("parses access_token and refresh_token from hash", () => {
      window.location.hash = "#access_token=acc&refresh_token=ref"
      const result = extractTokensFromHash()
      expect(result).toEqual({ access: "acc", refresh: "ref" })
    })

    it("clears the hash after parsing", () => {
      const replaceSpy = vi.spyOn(window.history, "replaceState")
      window.location.hash = "#access_token=acc&refresh_token=ref"
      extractTokensFromHash()
      expect(replaceSpy).toHaveBeenCalledWith(
        null,
        "",
        window.location.pathname
      )
    })

    it("returns null when only one token is in hash", () => {
      window.location.hash = "#access_token=acc"
      expect(extractTokensFromHash()).toBeNull()
    })
  })

  describe("getAccessToken", () => {
    it("returns null initially", () => {
      expect(getAccessToken()).toBeNull()
    })

    it("returns in-memory token after setTokens", () => {
      setTokens("my-token", "refresh")
      expect(getAccessToken()).toBe("my-token")
    })
  })

  describe("refreshAccessToken", () => {
    beforeEach(() => {
      vi.stubGlobal("fetch", vi.fn())
    })

    afterEach(() => {
      vi.unstubAllGlobals()
    })

    it("returns false when no refresh token in localStorage", async () => {
      // Given - no refresh token stored
      // When
      const result = await refreshAccessToken()
      // Then
      expect(result).toBe(false)
      expect(fetch).not.toHaveBeenCalled()
    })

    it("updates tokens and returns true on successful refresh", async () => {
      // Given
      localStorage.setItem("atlas_refresh_token", "old-refresh")
      vi.mocked(fetch).mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            access_token: "new-access",
            refresh_token: "new-refresh",
          }),
          { status: 200 },
        ),
      )
      // When
      const result = await refreshAccessToken()
      // Then
      expect(result).toBe(true)
      expect(getAccessToken()).toBe("new-access")
      expect(localStorage.getItem("atlas_refresh_token")).toBe("new-refresh")
    })

    it("calls clearAuth and returns false when server returns non-ok", async () => {
      // Given
      localStorage.setItem("atlas_refresh_token", "old-refresh")
      setTokens("old-access", "old-refresh")
      vi.mocked(fetch).mockResolvedValueOnce(
        new Response(null, { status: 401 }),
      )
      // When
      const result = await refreshAccessToken()
      // Then
      expect(result).toBe(false)
      expect(getAccessToken()).toBeNull()
      expect(localStorage.getItem("atlas_refresh_token")).toBeNull()
    })

    it("calls clearAuth and returns false on network error", async () => {
      // Given
      localStorage.setItem("atlas_refresh_token", "old-refresh")
      setTokens("old-access", "old-refresh")
      vi.mocked(fetch).mockRejectedValueOnce(new Error("Network failure"))
      // When
      const result = await refreshAccessToken()
      // Then
      expect(result).toBe(false)
      expect(getAccessToken()).toBeNull()
      expect(localStorage.getItem("atlas_refresh_token")).toBeNull()
    })
  })

  describe("apiFetch", () => {
    beforeEach(() => {
      vi.stubGlobal("fetch", vi.fn())
    })

    afterEach(() => {
      vi.unstubAllGlobals()
    })

    it("attaches Authorization header when access token exists", async () => {
      // Given
      setTokens("my-access-token", "my-refresh")
      vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status: 200 }))
      // When
      await apiFetch("/api/v1/some-endpoint")
      // Then
      const [, init] = vi.mocked(fetch).mock.calls[0]
      const headers = init?.headers as Headers
      expect(headers.get("Authorization")).toBe("Bearer my-access-token")
    })

    it("makes request without auth header when no token", async () => {
      // Given - no token set (clearAuth called in beforeEach)
      vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status: 200 }))
      // When
      await apiFetch("/api/v1/some-endpoint")
      // Then
      const [, init] = vi.mocked(fetch).mock.calls[0]
      const headers = init?.headers as Headers
      expect(headers.get("Authorization")).toBeNull()
    })

    it("retries with new token after 401 and successful refresh", async () => {
      // Given
      setTokens("expired-token", "valid-refresh")
      vi.mocked(fetch)
        // First call returns 401
        .mockResolvedValueOnce(new Response(null, { status: 401 }))
        // Refresh call succeeds
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({
              access_token: "new-access",
              refresh_token: "new-refresh",
            }),
            { status: 200 },
          ),
        )
        // Retry with new token succeeds
        .mockResolvedValueOnce(new Response(JSON.stringify({ ok: true }), { status: 200 }))
      // When
      const res = await apiFetch("/api/v1/protected")
      // Then
      expect(res.status).toBe(200)
      expect(vi.mocked(fetch)).toHaveBeenCalledTimes(3)
      const [, retryInit] = vi.mocked(fetch).mock.calls[2]
      const retryHeaders = retryInit?.headers as Headers
      expect(retryHeaders.get("Authorization")).toBe("Bearer new-access")
    })

    it("returns 401 response when refresh fails after 401", async () => {
      // Given
      setTokens("expired-token", "expired-refresh")
      vi.mocked(fetch)
        // First call returns 401
        .mockResolvedValueOnce(new Response(null, { status: 401 }))
        // Refresh call fails
        .mockResolvedValueOnce(new Response(null, { status: 401 }))
      // When
      const res = await apiFetch("/api/v1/protected")
      // Then
      expect(res.status).toBe(401)
      expect(vi.mocked(fetch)).toHaveBeenCalledTimes(2)
    })

    it("returns response as-is for non-401 errors", async () => {
      // Given
      setTokens("my-token", "my-refresh")
      vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status: 500 }))
      // When
      const res = await apiFetch("/api/v1/broken")
      // Then
      expect(res.status).toBe(500)
      expect(vi.mocked(fetch)).toHaveBeenCalledTimes(1)
    })
  })

  describe("fetchCurrentUser", () => {
    beforeEach(() => {
      vi.stubGlobal("fetch", vi.fn())
    })

    afterEach(() => {
      vi.unstubAllGlobals()
    })

    it("returns User object on successful response", async () => {
      // Given
      setTokens("valid-token", "valid-refresh")
      const mockUser = {
        id: "user-1",
        github_id: 42,
        login: "octocat",
        name: "The Octocat",
      }
      vi.mocked(fetch).mockResolvedValueOnce(
        new Response(JSON.stringify(mockUser), { status: 200 }),
      )
      // When
      const user = await fetchCurrentUser()
      // Then
      expect(user).toEqual(mockUser)
    })

    it("returns null on non-ok response", async () => {
      // Given
      setTokens("valid-token", "valid-refresh")
      vi.mocked(fetch).mockResolvedValueOnce(new Response(null, { status: 403 }))
      // When
      const user = await fetchCurrentUser()
      // Then
      expect(user).toBeNull()
    })

    it("returns null on network error", async () => {
      // Given
      setTokens("valid-token", "valid-refresh")
      vi.mocked(fetch).mockRejectedValueOnce(new Error("Network failure"))
      // When
      const user = await fetchCurrentUser()
      // Then
      expect(user).toBeNull()
    })
  })
})
