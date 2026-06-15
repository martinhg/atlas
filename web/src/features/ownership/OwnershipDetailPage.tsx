import { useParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { clearAuth } from "@/lib/auth";
import { useOwnershipDetail } from "./useOwnershipDetail";
import { OwnershipDetailTable } from "./OwnershipDetailTable";

interface Props {
  onLogout: () => void;
}

export function OwnershipDetailPage({ onLogout }: Props) {
  const { slug, repo } = useParams<{ slug: string; repo: string }>();

  const { data, isPending, isError } = useOwnershipDetail(slug!, repo!);

  const handleLogout = () => {
    clearAuth();
    onLogout();
  };

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link to="/dashboard" className="text-xl font-bold tracking-tight hover:text-zinc-300">
              Atlas
            </Link>
            <span className="text-zinc-600">/</span>
            <Link
              to={`/orgs/${slug}/ownership`}
              className="text-zinc-400 hover:text-zinc-200"
            >
              Ownership
            </Link>
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-300">{repo}</span>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleLogout}
            className="text-zinc-500 hover:text-zinc-300"
          >
            Sign out
          </Button>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8">
        <div className="space-y-6">
          <div>
            <h2 className="text-2xl font-semibold">{repo}</h2>
            <p className="text-zinc-500 text-sm mt-1">CODEOWNERS rules</p>
          </div>

          {isPending && (
            <p className="text-zinc-500 animate-pulse">Loading...</p>
          )}

          {isError && (
            <p className="text-red-400">Failed to load ownership details.</p>
          )}

          {data !== undefined && !isPending && !isError && (
            <OwnershipDetailTable rules={data.rules} />
          )}
        </div>
      </main>
    </div>
  );
}
