import { useQuery } from "@tanstack/react-query";
import { fetchRepoDeps } from "@/lib/api";

export function useRepoDeps(slug: string, name: string) {
  return useQuery({
    queryKey: ["repo-deps", slug, name],
    queryFn: () => fetchRepoDeps(slug, name),
    enabled: !!slug && !!name,
  });
}
