import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { GitHubCallbackPage } from "@/pages/GitHubCallbackPage";

vi.mock("@/lib/api", () => ({
  connectInstallation: vi.fn(),
}));

import { connectInstallation } from "@/lib/api";
const mockConnect = vi.mocked(connectInstallation);

function renderPage(search = "") {
  return render(
    <MemoryRouter initialEntries={[`/github/callback${search}`]}>
      <Routes>
        <Route path="/github/callback" element={<GitHubCallbackPage />} />
        <Route path="/orgs/:orgID/repos" element={<div>Repo List</div>} />
        <Route path="/dashboard" element={<div>Dashboard</div>} />
      </Routes>
    </MemoryRouter>
  );
}

describe("GitHubCallbackPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows error when installation_id is missing", async () => {
    renderPage();

    await waitFor(() => {
      expect(screen.getByText(/missing installation_id/i)).toBeInTheDocument();
    });
  });

  it("shows connecting state while processing", () => {
    mockConnect.mockReturnValue(new Promise(() => {}));
    renderPage("?installation_id=12345");

    expect(screen.getByText(/connecting github/i)).toBeInTheDocument();
  });

  it("redirects to repo list on success", async () => {
    mockConnect.mockResolvedValue({
      id: "org-uuid",
      github_id: 1,
      name: "my-org",
      slug: "my-org",
      owner_id: "u1",
      created_at: "",
      updated_at: "",
    });

    renderPage("?installation_id=12345");

    await waitFor(() => {
      expect(screen.getByText("Repo List")).toBeInTheDocument();
    });
    expect(mockConnect).toHaveBeenCalledWith(12345);
  });

  it("shows error on connection failure", async () => {
    mockConnect.mockRejectedValue(new Error("fail"));
    renderPage("?installation_id=99");

    await waitFor(() => {
      expect(screen.getByText(/failed to connect/i)).toBeInTheDocument();
    });
  });

  it("has back to dashboard link on error", async () => {
    mockConnect.mockRejectedValue(new Error("fail"));
    renderPage("?installation_id=99");

    await waitFor(() => {
      expect(screen.getByText(/back to dashboard/i)).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText(/back to dashboard/i));

    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
    });
  });
});
