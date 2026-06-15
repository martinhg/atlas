import { useParams, Link, useNavigate } from "react-router-dom";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { clearAuth } from "@/lib/auth";
import { useDependencies } from "./useDependencies";
import { DependencyTable } from "./DependencyTable";
import type { DependencyWithCount } from "@/lib/api";

interface Props {
  onLogout: () => void;
}

export function DependencyListPage({ onLogout }: Props) {
  const { slug } = useParams<{ slug: string }>();
  const [page, setPage] = useState(1);
  const perPage = 50;

  const { data, isPending, isError } = useDependencies(slug!, page, perPage);
  const navigate = useNavigate();

  const handleLogout = () => {
    clearAuth();
    onLogout();
  };

  const handleRowClick = (dep: DependencyWithCount) => {
    navigate(`/orgs/${slug}/dependencies/${dep.ecosystem}/${dep.name}`);
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
            <span className="text-zinc-400">Dependencies</span>
            <span className="text-zinc-600">·</span>
            <Link
              to={`/orgs/${slug}/repos`}
              className="text-zinc-500 hover:text-zinc-300 text-sm"
            >
              Repositories
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
            <h2 className="text-2xl font-semibold">Dependencies</h2>
            {data && (
              <span className="text-sm text-zinc-500">{total} total</span>
            )}
          </div>

          {isPending && (
            <p className="text-zinc-500 animate-pulse">Loading dependencies...</p>
          )}

          {isError && (
            <p className="text-red-400">Failed to load dependencies.</p>
          )}

          {data && (
            <DependencyTable
              deps={data.data}
              onRowClick={handleRowClick}
            />
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
