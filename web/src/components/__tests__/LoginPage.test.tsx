import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import LoginPage from "@/components/LoginPage"

describe("LoginPage", () => {
  it("renders the AtlasOS heading", () => {
    render(<LoginPage />)
    expect(screen.getByRole("heading", { name: "AtlasOS" })).toBeInTheDocument()
  })

  it("renders the Sign in with GitHub link", () => {
    render(<LoginPage />)
    expect(screen.getByRole("link", { name: /sign in with github/i })).toBeInTheDocument()
  })

  it("link points to the GitHub login endpoint", () => {
    render(<LoginPage />)
    const link = screen.getByRole("link", { name: /sign in with github/i })
    expect(link).toHaveAttribute("href", "/api/v1/auth/github/login")
  })

  it("renders the tagline text", () => {
    render(<LoginPage />)
    expect(
      screen.getByText(/map your software ecosystem/i)
    ).toBeInTheDocument()
  })
})
