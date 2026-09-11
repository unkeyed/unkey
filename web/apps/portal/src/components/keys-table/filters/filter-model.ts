import {
  OUTCOME_COLORS,
  OUTCOME_FILTER_KINDS,
  OUTCOME_FILTER_LABELS,
  type OutcomeFilterKind,
} from "~/components/analytics/outcomes";
import { KEY_STATUSES, KEY_STATUS_LABELS, keyStatus } from "../key-status";
import type { Key } from "../schema/keys.schema";

export type Dimension = "keys" | "outcomes" | "status";

export type FilterOption = {
  value: string;
  label: string;
  /** Requests in range, or keys with this status. Omitted when unavailable. */
  count?: number;
  /** Extra text the combined search matches, like a key's masked start. */
  search?: string;
  swatch?: string;
};

/**
 * One filterable dimension, built by the page so each dimension keeps its own
 * value type: the popover only ever reads labels and toggles by value.
 */
export type FilterDimension = {
  id: Dimension;
  label: string;
  /** Heading over this dimension's hits in the combined search. */
  groupLabel: string;
  /** Options with no requests hide behind a footer toggle. */
  hideEmpty?: boolean;
  options: FilterOption[];
  selected: string[];
  onToggle: (value: string) => void;
  onClear: () => void;
};

export type FilterBarProps = {
  dimensions: FilterDimension[];
};

export type AppliedFilter = {
  dimension: Dimension;
  label: string;
  value: string;
  onRemove: () => void;
};

export function appliedFilters(dimensions: FilterDimension[]): AppliedFilter[] {
  return dimensions.flatMap((dimension) => {
    if (dimension.selected.length === 0) {
      return [];
    }
    const value = dimension.selected
      .map((selected) => dimension.options.find((o) => o.value === selected)?.label ?? selected)
      .join(", ");
    return [
      {
        dimension: dimension.id,
        label: dimension.label,
        value,
        onRemove: dimension.onClear,
      },
    ];
  });
}

export function matchesText(needle: string, ...fields: Array<string | undefined>): boolean {
  const trimmed = needle.trim().toLowerCase();
  return trimmed === "" || fields.some((field) => field?.toLowerCase().includes(trimmed));
}

export function matchingOptions(options: FilterOption[], query: string): FilterOption[] {
  return options.filter((option) => matchesText(query, option.label, option.search));
}

export type SearchHit = {
  dimension: FilterDimension;
  options: FilterOption[];
};

export function searchHits(dimensions: FilterDimension[], query: string): SearchHit[] {
  return dimensions.map((dimension) => ({
    dimension,
    options: matchingOptions(dimension.options, query),
  }));
}

export type PanelOptions = {
  all: FilterOption[];
  visible: FilterOption[];
  empty: FilterOption[];
};

export function splitPanelOptions(
  panel: FilterDimension | undefined,
  panelQuery: string,
  showEmpty: boolean,
): PanelOptions {
  const all = panel ? matchingOptions(panel.options, panelQuery) : [];
  // An applied filter stays on its own row even once its count falls to zero,
  // so the only way to clear it never disappears.
  const hidable = (option: FilterOption) =>
    option.count === 0 && !panel?.selected.includes(option.value);
  return {
    all,
    visible: panel?.hideEmpty && !showEmpty ? all.filter((option) => !hidable(option)) : all,
    empty: panel?.hideEmpty ? all.filter(hidable) : [],
  };
}

export function toggle<T>(list: T[], value: T): T[] {
  return list.includes(value) ? list.filter((item) => item !== value) : [...list, value];
}

/** Toggles a value the popover handed back as a plain string, ignoring unknown ones. */
export function toggleFrom<T extends string>(
  all: ReadonlyArray<T>,
  selected: T[],
  value: string,
): T[] {
  const known = all.find((item) => item === value);
  return known === undefined ? selected : toggle(selected, known);
}

export function keyOptions(
  keys: ReadonlyArray<Key>,
  requests?: Map<string, number>,
): FilterOption[] {
  const options = keys.map((key) => ({
    value: key.id,
    label: key.name ?? key.start,
    search: key.start,
    count: requests?.get(key.id),
  }));
  return requests
    ? options.sort((a, b) => (b.count ?? 0) - (a.count ?? 0))
    : options.sort((a, b) => a.label.localeCompare(b.label));
}

export function outcomeOptions(counts: Record<OutcomeFilterKind, number>): FilterOption[] {
  return OUTCOME_FILTER_KINDS.map((kind) => ({
    value: kind,
    label: OUTCOME_FILTER_LABELS[kind],
    swatch: OUTCOME_COLORS[kind],
    count: counts[kind],
  }));
}

export function statusOptions(keys: ReadonlyArray<Key>): FilterOption[] {
  return KEY_STATUSES.map((status) => ({
    value: status,
    label: KEY_STATUS_LABELS[status],
    count: keys.filter((key) => keyStatus(key) === status).length,
  }));
}
