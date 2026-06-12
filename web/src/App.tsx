import { useEffect, useState } from "react";
import LoginPage from "./components/LoginPage";
import DashboardPage from "./components/DashboardPage";
import {
  type User,
  extractTokensFromHash,
  setTokens,
  hasRefreshToken,
  refreshAccessToken,
  fetchCurrentUser,
} from "./lib/auth";

type AuthStatus = "loading" | "unauthenticated" | "authenticated";

function App() {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    async function init() {
      const tokens = extractTokensFromHash();
      if (tokens) {
        setTokens(tokens.access, tokens.refresh);
        const me = await fetchCurrentUser();
        if (me) {
          setUser(me);
          setStatus("authenticated");
          return;
        }
      }

      if (hasRefreshToken()) {
        const refreshed = await refreshAccessToken();
        if (refreshed) {
          const me = await fetchCurrentUser();
          if (me) {
            setUser(me);
            setStatus("authenticated");
            return;
          }
        }
      }

      setStatus("unauthenticated");
    }

    init();
  }, []);

  if (status === "loading") {
    return (
      <div className="min-h-screen bg-zinc-950 text-zinc-100 flex items-center justify-center">
        <div className="text-zinc-500 animate-pulse">Loading...</div>
      </div>
    );
  }

  if (status === "authenticated" && user) {
    return (
      <DashboardPage
        user={user}
        onLogout={() => {
          setUser(null);
          setStatus("unauthenticated");
        }}
      />
    );
  }

  return <LoginPage />;
}

export default App;
