import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import type { SeverityLevel } from "@/lib/api";

interface Props {
  severity: SeverityLevel;
  className?: string;
}

const severityStyles: Record<SeverityLevel, string> = {
  critical: "bg-red-950 text-red-300 border-red-800",
  high: "bg-orange-950 text-orange-300 border-orange-800",
  medium: "bg-yellow-950 text-yellow-300 border-yellow-800",
  low: "bg-blue-950 text-blue-300 border-blue-800",
  unknown: "bg-zinc-800 text-zinc-400 border-zinc-700",
};

export function SeverityBadge({ severity, className }: Props) {
  return (
    <Badge
      variant="outline"
      className={cn("capitalize", severityStyles[severity], className)}
    >
      {severity}
    </Badge>
  );
}
