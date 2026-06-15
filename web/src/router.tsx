import { createBrowserRouter } from "react-router-dom";
import LoginPage from "@/components/LoginPage";
import { AuthGuard } from "@/components/auth/AuthGuard";
import DashboardPage from "@/components/DashboardPage";
import { GitHubCallbackPage } from "@/pages/GitHubCallbackPage";
import { RepoListPage } from "@/features/catalog/RepoListPage";
import { DependencyListPage } from "@/features/dependencies/DependencyListPage";
import { DependencyDetailPage } from "@/features/dependencies/DependencyDetailPage";
import { OwnershipListPage } from "@/features/ownership/OwnershipListPage";
import { OwnershipDetailPage } from "@/features/ownership/OwnershipDetailPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <LoginPage />,
  },
  {
    path: "/github/callback",
    element: <GitHubCallbackPage />,
  },
  {
    path: "/dashboard",
    element: (
      <AuthGuard>
        {(user, onLogout) => <DashboardPage user={user} onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
  {
    path: "/orgs/:slug/repos",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <RepoListPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
  {
    path: "/orgs/:slug/dependencies",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <DependencyListPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
  {
    path: "/orgs/:slug/dependencies/:ecosystem/*",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <DependencyDetailPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
  {
    path: "/orgs/:slug/ownership",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <OwnershipListPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
  {
    path: "/orgs/:slug/ownership/:repo",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <OwnershipDetailPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
]);
