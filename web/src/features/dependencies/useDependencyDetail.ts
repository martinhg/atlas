import { useQuery } from "@tanstack/react-query";
import { fetchDependencyDetail } from "@/lib/api";

export function useDependencyDetail(slug: string, ecosystem: string, name: string) {
  return useQuery({
    queryKey: ["dependency-detail", slug, ecosystem, name],
    queryFn: () => fetchDependencyDetail(slug, ecosystem, name),
    enabled: !!slug && !!ecosystem && !!name,
  });
}
