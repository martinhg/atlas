import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { GraphFilters } from "@/lib/api";
import { GraphFilters as GraphFiltersComponent } from "@/features/graph/GraphFilters";

describe("GraphFilters", () => {
  const onChange: (filters: GraphFilters) => void = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders ecosystem, risk, and team filter controls", () => {
    // Given / When
    render(
      <GraphFiltersComponent filters={{}} onChange={onChange} />,
    );

    // Then
    expect(screen.getByRole("combobox", { name: /ecosystem/i })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /low/i })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /medium/i })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: /high/i })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /team/i })).toBeInTheDocument();
  });

  it("calls onChange with ecosystem when ecosystem filter changes", async () => {
    // Given
    const user = userEvent.setup();
    render(
      <GraphFiltersComponent filters={{}} onChange={onChange} />,
    );

    // When
    await user.selectOptions(
      screen.getByRole("combobox", { name: /ecosystem/i }),
      "npm",
    );

    // Then
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ ecosystem: "npm" }));
  });

  it("calls onChange with risk when risk checkbox is toggled", async () => {
    // Given
    const user = userEvent.setup();
    render(
      <GraphFiltersComponent filters={{}} onChange={onChange} />,
    );

    // When
    await user.click(screen.getByRole("checkbox", { name: /high/i }));

    // Then
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ risk: "high" }));
  });

  it("calls onChange clearing risk when the active risk checkbox is unchecked", async () => {
    // Given
    const user = userEvent.setup();
    render(
      <GraphFiltersComponent filters={{ risk: "high" }} onChange={onChange} />,
    );

    // When — uncheck the checked checkbox
    await user.click(screen.getByRole("checkbox", { name: /high/i }));

    // Then
    expect(onChange).toHaveBeenCalledWith(expect.not.objectContaining({ risk: "high" }));
  });

  it("calls onChange with team value when team input changes", async () => {
    // Given — onChange is called once per keystroke; verify it passes team field
    const user = userEvent.setup();
    render(
      <GraphFiltersComponent filters={{}} onChange={onChange} />,
    );

    // When — type a single character
    await user.type(screen.getByRole("textbox", { name: /team/i }), "b");

    // Then — onChange should have been called with team: "b"
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ team: "b" }));
  });

  it("reflects active filter values in controls", () => {
    // Given / When
    render(
      <GraphFiltersComponent
        filters={{ ecosystem: "pypi", risk: "medium" }}
        onChange={onChange}
      />,
    );

    // Then
    const ecosystemSelect = screen.getByRole("combobox", { name: /ecosystem/i }) as HTMLSelectElement;
    expect(ecosystemSelect.value).toBe("pypi");
    expect(screen.getByRole("checkbox", { name: /medium/i })).toBeChecked();
  });
});
