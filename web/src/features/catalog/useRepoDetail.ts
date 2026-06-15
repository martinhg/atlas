import { useQuery } from "@tanstack/react-query";
import { fetchRepoDetail } from "@/lib/api";

export function useRepoDetail(slug: string, name: string) {
  return useQuery({
    queryKey: ["repo-detail", slug, name],
    queryFn: () => fetchRepoDetail(slug, name),
    enabled: !!slug && !!name,
  });
}
