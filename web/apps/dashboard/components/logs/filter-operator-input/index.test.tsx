import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FilterOperatorInput } from ".";

vi.mock("@unkey/icons", () => ({ Check: () => null }));
vi.mock("@unkey/ui", () => ({ Button: "button", Textarea: "textarea" }));

describe("FilterOperatorInput", () => {
  it("shows validation errors and only applies valid text", () => {
    const onApply = vi.fn();
    render(
      <FilterOperatorInput
        label="Path"
        options={[{ id: "contains", label: "contains" }]}
        validate={(_, value) => (value.length < 3 ? "Enter at least 3 characters." : null)}
        onApply={onApply}
      />,
    );

    const input = screen.getByPlaceholderText("Enter text");
    fireEvent.change(input, { target: { value: "ab" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    expect(screen.queryByText("Enter at least 3 characters.")).not.toBeNull();
    expect(input.getAttribute("aria-invalid")).toBe("true");
    expect(onApply).not.toHaveBeenCalled();

    fireEvent.change(input, { target: { value: "api" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    expect(onApply).toHaveBeenCalledWith("contains", "api");
  });
});
