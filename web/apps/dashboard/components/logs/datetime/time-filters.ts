import type { FilterValue } from "@/components/logs/validation/filter.types";

type TimeField = "startTime" | "endTime" | "since";

export type TimeFilter =
  | FilterValue<"startTime", "is", number>
  | FilterValue<"endTime", "is", number>
  | FilterValue<"since", "is", string>;

export type TimeValues = Partial<{ startTime: number; endTime: number; since: string }>;

const TIME_FIELDS: readonly TimeField[] = ["startTime", "endTime", "since"];

export function isTimeFilter<Filter extends FilterValue>(
  filter: Filter,
): filter is Filter & { field: TimeField } {
  return TIME_FIELDS.some((field) => field === filter.field);
}

export function timeValuesOf(filters: readonly FilterValue[]): TimeValues {
  const values: TimeValues = {};
  for (const { field, value } of filters) {
    if (field === "since" && typeof value === "string") {
      values.since = value;
    } else if (field === "startTime" && typeof value === "number") {
      values.startTime = value;
    } else if (field === "endTime" && typeof value === "number") {
      values.endTime = value;
    }
  }
  return values;
}

export function withTimeRange<Filter extends FilterValue>(
  filters: readonly Filter[],
  { startTime, endTime, since }: TimeValues,
): Array<Filter | TimeFilter> {
  const next: Array<Filter | TimeFilter> = filters.filter((filter) => !isTimeFilter(filter));
  if (since !== undefined) {
    next.push({ field: "since", value: since, id: crypto.randomUUID(), operator: "is" });
  } else if (startTime) {
    next.push({ field: "startTime", value: startTime, id: crypto.randomUUID(), operator: "is" });
    if (endTime) {
      next.push({ field: "endTime", value: endTime, id: crypto.randomUUID(), operator: "is" });
    }
  }
  return next;
}
