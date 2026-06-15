import { useQuery } from "@tanstack/react-query";
import { fetchDependencies } from "@/lib/api";

export function useDependencies(slug: string, page = 1, perPage = 50) {
  return useQuery({
    queryKey: ["dependencies", slug, page, perPage],
    queryFn: () => fetchDependencies(slug, page, perPage),
    enabled: !!slug,
  });
}
