import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import App from "@/App"
import * as auth from "@/lib/auth"

vi.mock("@/lib/api", () => ({
  fetchOrgs: vi.fn().mockResolvedValue([]),
}))

describe("App", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
    window.history.replaceState(null, "", "/")
  })

  it("shows LoginPage at root when unauthenticated", async () => {
    vi.spyOn(auth, "extractTokensFromHash").mockReturnValue(null)
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(false)

    render(<App />)

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Atlas" })).toBeInTheDocument()
    })
  })

  it("renders without crashing", () => {
    vi.spyOn(auth, "extractTokensFromHash").mockReturnValue(null)
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(false)

    const { container } = render(<App />)
    expect(container).toBeTruthy()
  })
})
