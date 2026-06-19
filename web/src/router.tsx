import { createBrowserRouter } from "react-router-dom";
import LoginPage from "@/components/LoginPage";
import { AuthGuard } from "@/components/auth/AuthGuard";
import DashboardPage from "@/components/DashboardPage";
import { GitHubCallbackPage } from "@/pages/GitHubCallbackPage";
import { RepoListPage } from "@/features/catalog/RepoListPage";
import { RepoDetailPage } from "@/features/catalog/RepoDetailPage";
import { DependencyListPage } from "@/features/dependencies/DependencyListPage";
import { DependencyDetailPage } from "@/features/dependencies/DependencyDetailPage";
import { OwnershipListPage } from "@/features/ownership/OwnershipListPage";
import { OwnershipDetailPage } from "@/features/ownership/OwnershipDetailPage";
import { ImpactAnalysisPage } from "@/features/impact/ImpactAnalysisPage";
import { VulnerabilityListPage } from "@/features/vulnerabilities/VulnerabilityListPage";
import { VulnerabilityDetailPage } from "@/features/vulnerabilities/VulnerabilityDetailPage";

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
    path: "/orgs/:slug/repos/:name",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <RepoDetailPage onLogout={onLogout} />}
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
  {
    path: "/orgs/:slug/impact",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <ImpactAnalysisPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
  {
    path: "/orgs/:slug/vulnerabilities",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <VulnerabilityListPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
  {
    path: "/orgs/:slug/vulnerabilities/:id",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <VulnerabilityDetailPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
]);
