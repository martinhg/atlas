import { useQuery } from "@tanstack/react-query";
import { fetchVulnerabilities } from "@/lib/api";

export function useVulnerabilities(
  slug: string,
  page = 1,
  perPage = 20,
  severity = "",
  packageName = "",
) {
  return useQuery({
    queryKey: ["vulnerabilities", slug, page, perPage, severity, packageName],
    queryFn: () => fetchVulnerabilities(slug, page, perPage, severity, packageName),
    enabled: !!slug,
  });
}
