import { useMutation } from "@tanstack/react-query";
import { analyzeImpact, type ImpactAnalysisRequest } from "@/lib/api";

export function useImpactAnalysis(slug: string) {
  return useMutation({
    mutationFn: (body: ImpactAnalysisRequest) => analyzeImpact(slug, body),
  });
}
