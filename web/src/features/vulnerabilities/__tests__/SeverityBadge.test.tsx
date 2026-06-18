import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { SeverityBadge } from "@/features/vulnerabilities/SeverityBadge";
import type { SeverityLevel } from "@/lib/api";

describe("SeverityBadge", () => {
  it.each<SeverityLevel>(["critical", "high", "medium", "low", "unknown"])(
    "renders the %s severity label",
    (severity) => {
      render(<SeverityBadge severity={severity} />);
      expect(screen.getByText(severity)).toBeInTheDocument();
    },
  );

  it("applies critical color styles", () => {
    render(<SeverityBadge severity="critical" />);
    expect(screen.getByText("critical").className).toContain("text-red-300");
  });
});
