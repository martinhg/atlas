import { useQuery } from "@tanstack/react-query";
import { fetchOrgs } from "@/lib/api";

export function useOrgs() {
  return useQuery({
    queryKey: ["orgs"],
    queryFn: fetchOrgs,
  });
}
