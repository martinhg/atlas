import { createBrowserRouter } from "react-router-dom";
import LoginPage from "@/components/LoginPage";
import { AuthGuard } from "@/components/auth/AuthGuard";
import DashboardPage from "@/components/DashboardPage";
import { GitHubCallbackPage } from "@/pages/GitHubCallbackPage";
import { RepoListPage } from "@/features/catalog/RepoListPage";

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
    path: "/orgs/:orgID/repos",
    element: (
      <AuthGuard>
        {(_user, onLogout) => <RepoListPage onLogout={onLogout} />}
      </AuthGuard>
    ),
  },
]);
