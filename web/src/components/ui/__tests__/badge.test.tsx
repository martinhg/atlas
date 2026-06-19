import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { Badge } from "@/components/ui/badge";

describe("Badge", () => {
  it("renders children", () => {
    render(<Badge>hello</Badge>);
    expect(screen.getByText("hello")).toBeInTheDocument();
  });

  it("applies default variant classes", () => {
    render(<Badge>default</Badge>);
    const el = screen.getByText("default");
    expect(el.className).toContain("inline-flex");
    expect(el.className).toContain("rounded-full");
  });

  it("applies outline variant classes", () => {
    render(<Badge variant="outline">outline</Badge>);
    const el = screen.getByText("outline");
    expect(el.className).toContain("border");
  });

  it("applies destructive variant classes", () => {
    render(<Badge variant="destructive">destructive</Badge>);
    const el = screen.getByText("destructive");
    expect(el.className).toContain("bg-destructive");
  });

  it("applies secondary variant classes", () => {
    render(<Badge variant="secondary">secondary</Badge>);
    const el = screen.getByText("secondary");
    expect(el.className).toContain("bg-secondary");
  });

  it("merges additional className", () => {
    render(<Badge className="text-red-300">custom</Badge>);
    const el = screen.getByText("custom");
    expect(el.className).toContain("text-red-300");
  });
});
