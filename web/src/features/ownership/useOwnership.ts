import { useQuery } from "@tanstack/react-query";
import { fetchOwnership } from "@/lib/api";

export function useOwnership(slug: string, page = 1, perPage = 50) {
  return useQuery({
    queryKey: ["ownership", slug, page, perPage],
    queryFn: () => fetchOwnership(slug, page, perPage),
    enabled: !!slug,
  });
}
