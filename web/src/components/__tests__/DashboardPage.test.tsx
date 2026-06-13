import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import DashboardPage from "@/components/DashboardPage"
import * as auth from "@/lib/auth"
import type { User } from "@/lib/auth"

vi.mock("@/hooks/useOrgs", () => ({
  useOrgs: () => ({ data: [], isLoading: false }),
}))

const baseUser: User = {
  id: "1",
  github_id: 42,
  login: "octocat",
  name: "The Octocat",
  avatar_url: "https://example.com/avatar.png",
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe("DashboardPage", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it("renders welcome message with user name", () => {
    renderWithProviders(<DashboardPage user={baseUser} onLogout={() => {}} />)
    expect(screen.getByText(/welcome, the octocat/i)).toBeInTheDocument()
  })

  it("renders avatar fallback with initials when avatar_url is provided", () => {
    renderWithProviders(<DashboardPage user={baseUser} onLogout={() => {}} />)
    expect(screen.getByText("OC")).toBeInTheDocument()
  })

  it("renders user login when no name is provided", () => {
    const user: User = { ...baseUser, name: undefined }
    renderWithProviders(<DashboardPage user={user} onLogout={() => {}} />)
    expect(screen.getByText(/welcome, octocat/i)).toBeInTheDocument()
  })

  it("sign out button calls onLogout", async () => {
    const onLogout = vi.fn()
    renderWithProviders(<DashboardPage user={baseUser} onLogout={onLogout} />)
    await userEvent.click(screen.getByRole("button", { name: /sign out/i }))
    expect(onLogout).toHaveBeenCalledOnce()
  })

  it("clearAuth is called on sign out", async () => {
    const clearAuthSpy = vi.spyOn(auth, "clearAuth")
    renderWithProviders(<DashboardPage user={baseUser} onLogout={() => {}} />)
    await userEvent.click(screen.getByRole("button", { name: /sign out/i }))
    expect(clearAuthSpy).toHaveBeenCalledOnce()
  })

  it("renders Connect GitHub button", () => {
    renderWithProviders(<DashboardPage user={baseUser} onLogout={() => {}} />)
    expect(screen.getByRole("link", { name: /connect github/i })).toBeInTheDocument()
  })

  it("shows organizations heading", () => {
    renderWithProviders(<DashboardPage user={baseUser} onLogout={() => {}} />)
    expect(screen.getByText("Organizations")).toBeInTheDocument()
  })
})
