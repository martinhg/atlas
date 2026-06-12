import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import App from "@/App"
import * as auth from "@/lib/auth"

describe("App", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it("shows loading state while async auth is in progress", () => {
    vi.spyOn(auth, "extractTokensFromHash").mockReturnValue(null)
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(true)
    // Never resolves — keeps App in loading state
    vi.spyOn(auth, "refreshAccessToken").mockReturnValue(new Promise(() => {}))

    render(<App />)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it("shows LoginPage when unauthenticated (no tokens)", async () => {
    vi.spyOn(auth, "extractTokensFromHash").mockReturnValue(null)
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(false)

    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "AtlasOS" })).toBeInTheDocument()
    })
  })

  it("shows LoginPage when refresh token exists but refresh fails", async () => {
    vi.spyOn(auth, "extractTokensFromHash").mockReturnValue(null)
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(true)
    vi.spyOn(auth, "refreshAccessToken").mockResolvedValue(false)

    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "AtlasOS" })).toBeInTheDocument()
    })
  })

  it("shows DashboardPage when tokens in hash resolve to a valid user", async () => {
    const mockUser: auth.User = {
      id: "1",
      github_id: 1,
      login: "octocat",
      name: "The Octocat",
    }
    vi.spyOn(auth, "extractTokensFromHash").mockReturnValue({
      access: "acc",
      refresh: "ref",
    })
    vi.spyOn(auth, "setTokens").mockImplementation(() => {})
    vi.spyOn(auth, "fetchCurrentUser").mockResolvedValue(mockUser)

    render(<App />)

    await waitFor(() => {
      expect(screen.getByText(/welcome, the octocat/i)).toBeInTheDocument()
    })
  })
})
