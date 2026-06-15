import { useParams, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { clearAuth } from "@/lib/auth";
import { useDependencyDetail } from "./useDependencyDetail";
import { DependencyDetailTable } from "./DependencyDetailTable";

interface Props {
  onLogout: () => void;
}

export function DependencyDetailPage({ onLogout }: Props) {
  const { slug, ecosystem, name, "*": splat } = useParams<{
    slug: string;
    ecosystem: string;
    name?: string;
    "*"?: string;
  }>();

  // Support scoped packages: when using wildcard route, `splat` holds the full name
  // e.g. /orgs/:slug/dependencies/npm/* → splat = "@scope/pkg"
  const pkgName = splat ?? name ?? "";

  const { data, isPending, isError } = useDependencyDetail(slug!, ecosystem!, pkgName);

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
              to={`/orgs/${slug}/dependencies`}
              className="text-zinc-400 hover:text-zinc-200"
            >
              Dependencies
            </Link>
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-300">{pkgName}</span>
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
            <h2 className="text-2xl font-semibold">{pkgName}</h2>
            <p className="text-zinc-500 text-sm mt-1">{ecosystem}</p>
          </div>

          {isPending && (
            <p className="text-zinc-500 animate-pulse">Loading...</p>
          )}

          {isError && (
            <p className="text-red-400">Failed to load dependency details.</p>
          )}

          {data !== undefined && !isPending && !isError && (
            <DependencyDetailTable repos={data} />
          )}
        </div>
      </main>
    </div>
  );
}
