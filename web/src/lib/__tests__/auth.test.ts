import { describe, it, expect, beforeEach, vi } from "vitest"
import {
  setTokens,
  getAccessToken,
  clearAuth,
  hasRefreshToken,
  extractTokensFromHash,
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
})
