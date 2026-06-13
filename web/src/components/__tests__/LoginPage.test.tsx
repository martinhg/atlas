import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import LoginPage from "@/components/LoginPage"
import * as auth from "@/lib/auth"

function renderWithRouter(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>)
}

describe("LoginPage", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.spyOn(auth, "extractTokensFromHash").mockReturnValue(null)
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(false)
  })

  it("renders the Atlas heading", () => {
    renderWithRouter(<LoginPage />)
    expect(screen.getByRole("heading", { name: "Atlas" })).toBeInTheDocument()
  })

  it("renders the Sign in with GitHub link", () => {
    renderWithRouter(<LoginPage />)
    expect(screen.getByRole("link", { name: /sign in with github/i })).toBeInTheDocument()
  })

  it("link points to the GitHub login endpoint", () => {
    renderWithRouter(<LoginPage />)
    const link = screen.getByRole("link", { name: /sign in with github/i })
    expect(link).toHaveAttribute("href", "/api/v1/auth/github/login")
  })

  it("renders the tagline text", () => {
    renderWithRouter(<LoginPage />)
    expect(
      screen.getByText(/map your software ecosystem/i)
    ).toBeInTheDocument()
  })
})
