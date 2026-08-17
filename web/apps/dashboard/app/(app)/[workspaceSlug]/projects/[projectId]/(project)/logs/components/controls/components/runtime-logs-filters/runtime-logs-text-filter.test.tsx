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
  it("applies exact attribute matches", () => {
    render(<RuntimeLogsTextFilter field="attributes" label="Attributes" />);

    fireEvent.click(screen.getByRole("button", { name: "matches" }));
    const input = screen.getByPlaceholderText("request.id = xyz");
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
