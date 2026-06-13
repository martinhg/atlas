import { useEffect, useState } from "react";
import { Navigate } from "react-router-dom";
import {
  type User,
  hasRefreshToken,
  refreshAccessToken,
  fetchCurrentUser,
} from "@/lib/auth";

interface AuthGuardProps {
  children: (user: User, onLogout: () => void) => React.ReactNode;
}

export function AuthGuard({ children }: AuthGuardProps) {
  const [status, setStatus] = useState<"loading" | "authenticated" | "unauthenticated">("loading");
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    async function checkAuth() {
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
    checkAuth();
  }, []);

  const handleLogout = () => {
    setUser(null);
    setStatus("unauthenticated");
  };

  if (status === "loading") {
    return (
      <div className="min-h-screen bg-zinc-950 text-zinc-100 flex items-center justify-center">
        <div className="text-zinc-500 animate-pulse">Loading...</div>
      </div>
    );
  }

  if (status === "unauthenticated") {
    return <Navigate to="/" replace />;
  }

  return <>{children(user!, handleLogout)}</>;
}
