import { useParams, Link } from "react-router-dom";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { clearAuth } from "@/lib/auth";
import { useOwnership } from "./useOwnership";
import { OwnershipTable } from "./OwnershipTable";

interface Props {
  onLogout: () => void;
}

export function OwnershipListPage({ onLogout }: Props) {
  const { slug } = useParams<{ slug: string }>();
  const [page, setPage] = useState(1);
  const perPage = 50;

  const { data, isPending, isError } = useOwnership(slug!, page, perPage);

  const handleLogout = () => {
    clearAuth();
    onLogout();
  };

  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / perPage);

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link to="/dashboard" className="text-xl font-bold tracking-tight hover:text-zinc-300">
              Atlas
            </Link>
            <span className="text-zinc-600">/</span>
            <span className="text-zinc-400">Ownership</span>
            <span className="text-zinc-600">·</span>
            <Link
              to={`/orgs/${slug}/repos`}
              className="text-zinc-500 hover:text-zinc-300 text-sm"
            >
              Repositories
            </Link>
            <Link
              to={`/orgs/${slug}/dependencies`}
              className="text-zinc-500 hover:text-zinc-300 text-sm"
            >
              Dependencies
            </Link>
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
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">Ownership</h2>
            {data && (
              <span className="text-sm text-zinc-500">{total} total</span>
            )}
          </div>

          {isPending && (
            <p className="text-zinc-500 animate-pulse">Loading ownership...</p>
          )}

          {isError && (
            <p className="text-red-400">Failed to load ownership.</p>
          )}

          {data && (
            <OwnershipTable data={data.data} slug={slug!} />
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-4">
              <Button
                variant="ghost"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="text-zinc-400 hover:text-zinc-200"
              >
                Previous
              </Button>
              <span className="text-sm text-zinc-500">
                Page {page} of {totalPages}
              </span>
              <Button
                variant="ghost"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="text-zinc-400 hover:text-zinc-200"
              >
                Next
              </Button>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
