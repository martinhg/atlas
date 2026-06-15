import { useQuery } from "@tanstack/react-query";
import { fetchRepos } from "@/lib/api";

export function useRepos(slug: string) {
  return useQuery({
    queryKey: ["repos", slug],
    queryFn: () => fetchRepos(slug),
    enabled: !!slug,
  });
}
