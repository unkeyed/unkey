import { describe, expect, it, vi } from "vitest";
import type { FilterValue } from "../validation/filter.types";
import { type FilterItemConfig, createOptionSearchItems } from "./filters-popover";

type TestField = "status" | "bucket" | "region";
type TestFilter = FilterValue<TestField, "is" | "contains", string | number>;

const filter = (field: TestField, value: string | number): TestFilter => ({
  id: `${field}-${value}`,
  field,
  operator: "is",
  value,
});

const items: FilterItemConfig<TestField>[] = [
  {
    id: "status",
    label: "Status",
    component: null,
    options: [200, 404, 500].map((value) => ({ value, label: String(value) })),
  },
  {
    id: "bucket",
    label: "Bucket",
    component: null,
    single: true,
    options: ["a", "b"].map((value) => ({ value, label: value })),
  },
  { id: "region", label: "Region", component: null },
];

const select = (filters: TestFilter[], id: string) => {
  const updateFilters = vi.fn<(filters: TestFilter[]) => void>();
  const item = createOptionSearchItems(items, filters, updateFilters).find((i) => i.id === id);
  if (item?.kind !== "option") {
    throw new Error(`expected option item ${id}`);
  }
  item.onSelect();
  return { item, next: updateFilters.mock.calls[0]?.[0] ?? [] };
};

const values = (filters: TestFilter[]) => filters.map((f) => `${f.field}=${f.value}`).sort();

describe("createOptionSearchItems", () => {
  const current = [filter("status", 200), filter("status", 404), filter("region", "eu")];

  it("creates one option per item value under the item label", () => {
    const options = createOptionSearchItems(items, current, vi.fn());
    expect(options.map((o) => [o.id, o.path.join("/")])).toEqual([
      ["status:200", "Status"],
      ["status:404", "Status"],
      ["status:500", "Status"],
      ["bucket:a", "Bucket"],
      ["bucket:b", "Bucket"],
    ]);
  });

  it("adds an unselected value as an is filter and keeps other values of the same field", () => {
    const { item, next } = select(current, "status:500");
    expect(item.checked).toBe(false);
    expect(values(next)).toEqual(["region=eu", "status=200", "status=404", "status=500"]);
    expect(next.find((f) => f.value === 500)?.value).toBe(500);
  });

  it("removes only the selected value", () => {
    const { item, next } = select(current, "status:404");
    expect(item.checked).toBe(true);
    expect(values(next)).toEqual(["region=eu", "status=200"]);
  });

  it("matches a numeric option against a value parsed from the URL as a string", () => {
    const { item, next } = select([filter("status", "500")], "status:500");
    expect(item.checked).toBe(true);
    expect(next).toEqual([]);
  });

  it("replaces the current value of a single-value field", () => {
    const { next } = select([...current, filter("bucket", "a")], "bucket:b");
    expect(values(next)).toEqual(["bucket=b", "region=eu", "status=200", "status=404"]);
  });

  it("clears a single-value field when its selected value is chosen again", () => {
    const { item, next } = select([...current, filter("bucket", "a")], "bucket:a");
    expect(item.checked).toBe(true);
    expect(values(next)).toEqual(["region=eu", "status=200", "status=404"]);
  });
});
