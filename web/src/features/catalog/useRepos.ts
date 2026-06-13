import { useQuery } from "@tanstack/react-query";
import { fetchRepos } from "@/lib/api";

export function useRepos(orgID: string) {
  return useQuery({
    queryKey: ["repos", orgID],
    queryFn: () => fetchRepos(orgID),
    enabled: !!orgID,
  });
}
