import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { AuthGuard } from "@/components/auth/AuthGuard";
import * as auth from "@/lib/auth";

function renderWithRouter(ui: React.ReactElement, initialRoute = "/protected") {
  return render(
    <MemoryRouter initialEntries={[initialRoute]}>
      <Routes>
        <Route path="/" element={<div>Login Page</div>} />
        <Route path="/protected" element={ui} />
      </Routes>
    </MemoryRouter>
  );
}

describe("AuthGuard", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("shows loading state initially when refresh token exists", () => {
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(true);
    vi.spyOn(auth, "refreshAccessToken").mockReturnValue(new Promise(() => {}));

    renderWithRouter(
      <AuthGuard>{(user) => <div>Hello {user.login}</div>}</AuthGuard>
    );

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it("redirects to / when no refresh token", async () => {
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(false);

    renderWithRouter(
      <AuthGuard>{(user) => <div>Hello {user.login}</div>}</AuthGuard>
    );

    await waitFor(() => {
      expect(screen.getByText("Login Page")).toBeInTheDocument();
    });
  });

  it("redirects to / when refresh fails", async () => {
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(true);
    vi.spyOn(auth, "refreshAccessToken").mockResolvedValue(false);

    renderWithRouter(
      <AuthGuard>{(user) => <div>Hello {user.login}</div>}</AuthGuard>
    );

    await waitFor(() => {
      expect(screen.getByText("Login Page")).toBeInTheDocument();
    });
  });

  it("redirects to / when refresh succeeds but fetchCurrentUser fails", async () => {
    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(true);
    vi.spyOn(auth, "refreshAccessToken").mockResolvedValue(true);
    vi.spyOn(auth, "fetchCurrentUser").mockResolvedValue(null);

    renderWithRouter(
      <AuthGuard>{(user) => <div>Hello {user.login}</div>}</AuthGuard>
    );

    await waitFor(() => {
      expect(screen.getByText("Login Page")).toBeInTheDocument();
    });
  });

  it("renders children with user when authenticated", async () => {
    const mockUser: auth.User = {
      id: "1",
      github_id: 42,
      login: "octocat",
      name: "The Octocat",
    };

    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(true);
    vi.spyOn(auth, "refreshAccessToken").mockResolvedValue(true);
    vi.spyOn(auth, "fetchCurrentUser").mockResolvedValue(mockUser);

    renderWithRouter(
      <AuthGuard>{(user) => <div>Hello {user.login}</div>}</AuthGuard>
    );

    await waitFor(() => {
      expect(screen.getByText("Hello octocat")).toBeInTheDocument();
    });
  });

  it("passes onLogout that resets to unauthenticated", async () => {
    const mockUser: auth.User = {
      id: "1",
      github_id: 42,
      login: "octocat",
    };

    vi.spyOn(auth, "hasRefreshToken").mockReturnValue(true);
    vi.spyOn(auth, "refreshAccessToken").mockResolvedValue(true);
    vi.spyOn(auth, "fetchCurrentUser").mockResolvedValue(mockUser);

    renderWithRouter(
      <AuthGuard>
        {(user, onLogout) => (
          <div>
            <span>Hello {user.login}</span>
            <button onClick={onLogout}>Logout</button>
          </div>
        )}
      </AuthGuard>
    );

    await waitFor(() => {
      expect(screen.getByText("Hello octocat")).toBeInTheDocument();
    });

    const { default: userEvent } = await import("@testing-library/user-event");
    await userEvent.click(screen.getByRole("button", { name: /logout/i }));

    await waitFor(() => {
      expect(screen.getByText("Login Page")).toBeInTheDocument();
    });
  });
});
