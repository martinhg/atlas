import { useQuery } from "@tanstack/react-query";
import { fetchRepos } from "@/lib/api";

export function useRepos(slug: string, page = 1, perPage = 25, q = "") {
  return useQuery({
    queryKey: ["repos", slug, page, perPage, q],
    queryFn: () => fetchRepos(slug, page, perPage, q),
    enabled: !!slug,
  });
}
