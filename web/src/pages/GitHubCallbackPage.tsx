import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { connectInstallation } from "@/lib/api";

export function GitHubCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function handleCallback() {
      const installationId = searchParams.get("installation_id");
      if (!installationId) {
        setError("Missing installation_id parameter");
        return;
      }

      try {
        const org = await connectInstallation(Number(installationId));
        navigate(`/orgs/${org.slug}/repos`, { replace: true });
      } catch {
        setError("Failed to connect GitHub installation. Please try again.");
      }
    }
    handleCallback();
  }, [searchParams, navigate]);

  if (error) {
    return (
      <div className="min-h-screen bg-zinc-950 text-zinc-100 flex items-center justify-center">
        <div className="text-center space-y-4">
          <p className="text-red-400">{error}</p>
          <button
            onClick={() => navigate("/dashboard")}
            className="text-zinc-400 hover:text-zinc-200 underline"
          >
            Back to dashboard
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 flex items-center justify-center">
      <div className="text-zinc-500 animate-pulse">Connecting GitHub...</div>
    </div>
  );
}
