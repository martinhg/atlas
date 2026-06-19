import { useQuery } from "@tanstack/react-query";
import { fetchGraphData, type GraphFilters } from "@/lib/api";

export function useGraphData(slug: string, filters: GraphFilters) {
  return useQuery({
    queryKey: ["graph", slug, filters],
    queryFn: () => fetchGraphData(slug, filters),
    enabled: !!slug,
  });
}
