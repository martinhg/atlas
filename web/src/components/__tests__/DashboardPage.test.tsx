import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import DashboardPage from "@/components/DashboardPage"
import * as auth from "@/lib/auth"
import type { User } from "@/lib/auth"

const baseUser: User = {
  id: "1",
  github_id: 42,
  login: "octocat",
  name: "The Octocat",
  avatar_url: "https://example.com/avatar.png",
}

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it("renders welcome message with user name", () => {
    render(<DashboardPage user={baseUser} onLogout={() => {}} />)
    expect(screen.getByText(/welcome, the octocat/i)).toBeInTheDocument()
  })

  it("renders avatar fallback with initials when avatar_url is provided", () => {
    render(<DashboardPage user={baseUser} onLogout={() => {}} />)
    // Radix AvatarImage only shows after load event; jsdom never fires it.
    // The AvatarFallback renders with the user's initials instead.
    expect(screen.getByText("OC")).toBeInTheDocument()
  })

  it("renders user login when no name is provided", () => {
    const user: User = { ...baseUser, name: undefined }
    render(<DashboardPage user={user} onLogout={() => {}} />)
    expect(screen.getByText(/welcome, octocat/i)).toBeInTheDocument()
  })

  it("sign out button calls onLogout", async () => {
    const onLogout = vi.fn()
    render(<DashboardPage user={baseUser} onLogout={onLogout} />)
    await userEvent.click(screen.getByRole("button", { name: /sign out/i }))
    expect(onLogout).toHaveBeenCalledOnce()
  })

  it("clearAuth is called on sign out", async () => {
    const clearAuthSpy = vi.spyOn(auth, "clearAuth")
    render(<DashboardPage user={baseUser} onLogout={() => {}} />)
    await userEvent.click(screen.getByRole("button", { name: /sign out/i }))
    expect(clearAuthSpy).toHaveBeenCalledOnce()
  })
})
