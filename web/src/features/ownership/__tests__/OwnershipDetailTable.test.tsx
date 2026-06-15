import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { OwnershipDetailTable } from "@/features/ownership/OwnershipDetailTable";
import type { OwnerRule } from "@/lib/api";

const makeRule = (overrides: Partial<OwnerRule> = {}): OwnerRule => ({
  pattern: "*.go",
  owner: "@org/backend",
  owner_type: "team",
  line_number: 5,
  ...overrides,
});

describe("OwnershipDetailTable", () => {
  it("shows empty state when no rules", () => {
    render(<OwnershipDetailTable rules={[]} />);
    expect(screen.getByText(/no codeowners rules found for this repository/i)).toBeInTheDocument();
  });

  it("renders column headers", () => {
    render(<OwnershipDetailTable rules={[makeRule()]} />);
    expect(screen.getByText("Pattern")).toBeInTheDocument();
    expect(screen.getByText("Owner")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
    expect(screen.getByText("Line")).toBeInTheDocument();
  });

  it("renders pattern, owner, type and line number", () => {
    render(<OwnershipDetailTable rules={[makeRule({ pattern: "src/api/**", owner: "@org/backend", owner_type: "team", line_number: 3 })]} />);
    expect(screen.getByText("src/api/**")).toBeInTheDocument();
    expect(screen.getByText("@org/backend")).toBeInTheDocument();
    expect(screen.getByText("team")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("shows dash when line_number is absent", () => {
    render(<OwnershipDetailTable rules={[makeRule({ line_number: undefined })]} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("renders user owner_type badge", () => {
    render(<OwnershipDetailTable rules={[makeRule({ owner: "@username", owner_type: "user" })]} />);
    expect(screen.getByText("user")).toBeInTheDocument();
  });

  it("renders email owner_type badge", () => {
    render(<OwnershipDetailTable rules={[makeRule({ owner: "user@example.com", owner_type: "email" })]} />);
    expect(screen.getByText("email")).toBeInTheDocument();
  });

  it("renders multiple rows", () => {
    const rules = [
      makeRule({ pattern: "*.go", owner: "@org/backend" }),
      makeRule({ pattern: "*.ts", owner: "@org/frontend" }),
    ];
    render(<OwnershipDetailTable rules={rules} />);
    expect(screen.getByText("*.go")).toBeInTheDocument();
    expect(screen.getByText("*.ts")).toBeInTheDocument();
    expect(screen.getByText("@org/backend")).toBeInTheDocument();
    expect(screen.getByText("@org/frontend")).toBeInTheDocument();
  });
});
