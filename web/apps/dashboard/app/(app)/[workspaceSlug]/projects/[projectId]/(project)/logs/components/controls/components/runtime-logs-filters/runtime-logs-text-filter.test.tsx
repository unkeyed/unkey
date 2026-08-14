import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RuntimeLogsTextFilter } from "./runtime-logs-text-filter";

const { updateFilters } = vi.hoisted(() => ({ updateFilters: vi.fn() }));

vi.mock(
  "@/app/(app)/[workspaceSlug]/projects/[projectId]/(project)/logs/hooks/use-runtime-logs-filters",
  () => ({
    useRuntimeLogsFilters: () => ({ filters: [], updateFilters }),
  }),
);
vi.mock("@unkey/icons", () => ({ Check: () => null }));
vi.mock("@unkey/ui", () => ({ Button: "button", Textarea: "textarea" }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("RuntimeLogsTextFilter", () => {
  it("requires indexed attribute searches and applies the attribute field", () => {
    render(<RuntimeLogsTextFilter field="attributes" label="Attributes" />);

    const input = screen.getByPlaceholderText("Enter attribute text");
    fireEvent.change(input, { target: { value: "ab" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    expect(screen.queryByText("Enter at least 3 characters.")).not.toBeNull();
    expect(updateFilters).not.toHaveBeenCalled();

    fireEvent.change(input, { target: { value: "tenant_id" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    expect(updateFilters).toHaveBeenCalledWith([
      expect.objectContaining({
        field: "attributes",
        operator: "contains",
        value: "tenant_id",
      }),
    ]);
  });

  it("validates and applies exact attribute matches", () => {
    render(<RuntimeLogsTextFilter field="attributes" label="Attributes" />);

    fireEvent.click(screen.getByRole("button", { name: "matches" }));
    const input = screen.getByPlaceholderText("request.id = xyz");
    fireEvent.change(input, { target: { value: "request.id" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    expect(screen.queryByText("Use path = value with a value of at least 3 characters.")).not.toBe(
      null,
    );
    expect(updateFilters).not.toHaveBeenCalled();

    fireEvent.change(input, { target: { value: "request.id = token=xyz" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    expect(updateFilters).toHaveBeenCalledWith([
      expect.objectContaining({
        field: "attributes",
        operator: "is",
        value: "request.id = token=xyz",
      }),
    ]);
  });
});
