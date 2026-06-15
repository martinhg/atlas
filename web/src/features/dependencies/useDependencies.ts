import { useQuery } from "@tanstack/react-query";
import { fetchDependencies } from "@/lib/api";

export function useDependencies(slug: string, page = 1, perPage = 50, q = "") {
  return useQuery({
    queryKey: ["dependencies", slug, page, perPage, q],
    queryFn: () => fetchDependencies(slug, page, perPage, q),
    enabled: !!slug,
  });
}
