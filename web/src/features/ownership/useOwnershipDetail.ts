import { useQuery } from "@tanstack/react-query";
import { fetchOwnershipDetail } from "@/lib/api";

export function useOwnershipDetail(slug: string, repo: string) {
  return useQuery({
    queryKey: ["ownership-detail", slug, repo],
    queryFn: () => fetchOwnershipDetail(slug, repo),
    enabled: !!slug && !!repo,
  });
}
