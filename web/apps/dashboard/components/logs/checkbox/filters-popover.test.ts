import { describe, expect, it, vi } from "vitest";
import type { FilterValue } from "../validation/filter.types";
import { createFilterOptionItem } from "./filters-popover";

type TestFilter = FilterValue<"status" | "bucket" | "region", "is">;

const filter = (field: TestFilter["field"], value: string): TestFilter => ({
  id: `${field}-${value}`,
  field,
  operator: "is",
  value,
});

const select = (
  filters: TestFilter[],
  field: TestFilter["field"],
  value: string,
  single?: boolean,
) => {
  const updateFilters = vi.fn<(filters: TestFilter[]) => void>();
  const item = createFilterOptionItem({
    filters,
    updateFilters,
    createFilter: filter,
    field,
    value,
    single,
    label: value,
    path: [],
  });
  if (item.kind !== "option") {
    throw new Error("expected an option item");
  }
  item.onSelect();
  return { checked: item.checked, next: updateFilters.mock.calls[0]?.[0] ?? [] };
};

const values = (filters: TestFilter[]) => filters.map((f) => `${f.field}=${f.value}`).sort();

describe("createFilterOptionItem", () => {
  const current = [filter("status", "200"), filter("status", "404"), filter("region", "eu")];

  it("adds an unselected value and keeps other values of the same field", () => {
    const { checked, next } = select(current, "status", "500");
    expect(checked).toBe(false);
    expect(values(next)).toEqual(["region=eu", "status=200", "status=404", "status=500"]);
  });

  it("removes only the selected value", () => {
    const { checked, next } = select(current, "status", "404");
    expect(checked).toBe(true);
    expect(values(next)).toEqual(["region=eu", "status=200"]);
  });

  it("replaces the current value of a single-value field", () => {
    const { next } = select([...current, filter("bucket", "a")], "bucket", "b", true);
    expect(values(next)).toEqual(["bucket=b", "region=eu", "status=200", "status=404"]);
  });

  it("clears a single-value field when its selected value is chosen again", () => {
    const { checked, next } = select([...current, filter("bucket", "a")], "bucket", "a", true);
    expect(checked).toBe(true);
    expect(values(next)).toEqual(["region=eu", "status=200", "status=404"]);
  });
});
