import {
  type VerificationMetrics,
  computeMetrics,
} from "~/components/analytics/analytics-transform";
import type { ChartState } from "~/components/analytics/chart-slot";
import {
  type FilteredUsage,
  type KeyUsage,
  applyFilters,
  sumSeries,
} from "~/components/analytics/key-usage";
import { OUTCOME_FILTER_KINDS, type OutcomeFilterKind } from "~/components/analytics/outcomes";
import type { VerificationBucket } from "~/components/analytics/schema/analytics.schema";
import type { TimePreset, TimePresetId } from "~/components/analytics/time-presets";
import {
  type FilterBarProps,
  type FilterDimension,
  keyOptions,
  outcomeOptions,
  statusOptions,
  toggle,
  toggleFrom,
} from "~/components/keys-table/filters/filter-model";
import type { KeyRow } from "~/components/keys-table/key-row";
import { KEY_STATUSES, type KeyStatus, keyStatus } from "~/components/keys-table/key-status";
import type { Key } from "~/components/keys-table/schema/keys.schema";
import type { KeysSearchPatch } from "~/hooks/use-keys-search";
import type { KeyUsageRow, KeysUsage } from "~/hooks/use-keys-usage";
import { RETENTION_EXCEEDED_MESSAGE, isRetentionExceededError } from "~/lib/portal-api";

export type KeysPageModelInput = {
  usage: KeysUsage;
  selectedKeys: string[];
  selectedOutcomes: OutcomeFilterKind[];
  selectedStatus: KeyStatus[];
};

export type KeysPageModel = {
  /** The selection the page actually renders, which the key list can override. */
  selectedKeys: string[];
  /** The keys left after the key and status filters, in list order. */
  effectiveKeys: Key[];
  filtered: FilteredUsage;
  metrics: VerificationMetrics;
  chart: ChartState;
  requestsByKey: Map<string, number> | undefined;
  missingFromTotals: boolean;
};

function settledUsage(
  keys: ReadonlyArray<Key>,
  rows: ReadonlyMap<string, KeyUsageRow> | null,
): KeyUsage[] {
  return keys.flatMap((key) => {
    const row = rows?.get(key.id);
    return row?.status === "ok" ? [row.usage] : [];
  });
}

function buildChart(
  input: KeysPageModelInput,
  narrowed: boolean,
  missingFromTotals: boolean,
  totalRequests: number,
): ChartState {
  const { aggregateState, keysState, byKey } = input.usage;

  if (
    aggregateState.isInitialLoading ||
    (narrowed && (keysState.isInitialLoading || (byKey !== "unavailable" && byKey.isPending)))
  ) {
    return { kind: "loading" };
  }
  if (aggregateState.isError) {
    return isRetentionExceededError(aggregateState.error)
      ? { kind: "error", message: RETENTION_EXCEEDED_MESSAGE }
      : {
          kind: "error",
          message: "Couldn't load your analytics",
          onRetry: aggregateState.refetch,
        };
  }
  if (narrowed && byKey === "unavailable") {
    return { kind: "error", message: "Usage per key isn't available for this account." };
  }
  if (missingFromTotals && byKey !== "unavailable") {
    return {
      kind: "error",
      message: "Couldn't load usage for some of these keys",
      onRetry: byKey.retryFailed,
    };
  }
  return totalRequests === 0 ? { kind: "empty" } : { kind: "populated" };
}

/**
 * The chart reads the account aggregate whenever the key and status filters
 * leave every key in play: an outcome filter alone only projects those buckets,
 * so the page never waits on — or is shortened by — the per-key requests. Only
 * a narrower key set rebuilds the series from them.
 */
export function deriveKeysPageModel(input: KeysPageModelInput): KeysPageModel {
  const { selectedOutcomes, selectedStatus, usage } = input;
  const { keys, keysState, aggregate, byKey, analytics } = usage;

  // An unreadable key list leaves nothing to select, so the chart falls back to
  // the whole account instead of an empty selection.
  const selectedKeys = keysState.isError ? [] : input.selectedKeys;
  const effectiveKeys = keys.filter(
    (key) =>
      (selectedStatus.length === 0 || selectedStatus.includes(keyStatus(key))) &&
      (selectedKeys.length === 0 || selectedKeys.includes(key.id)),
  );
  const narrowed = effectiveKeys.length !== keys.length;

  const rows = byKey === "unavailable" ? null : byKey.rows;
  const inScope = settledUsage(effectiveKeys, rows);
  const filtered = applyFilters(
    inScope,
    narrowed ? sumSeries(inScope) : aggregate,
    selectedOutcomes,
  );
  const metrics = computeMetrics(filtered.totals);

  // A narrowed chart is rebuilt from the per-key series, so a key that failed to
  // load would quietly subtract itself from it. Report that instead of drawing a
  // chart that is confidently short.
  const missingFromTotals =
    narrowed && effectiveKeys.some((key) => rows?.get(key.id)?.status === "error");

  return {
    selectedKeys,
    effectiveKeys,
    filtered,
    metrics,
    missingFromTotals,
    chart: buildChart(input, narrowed, missingFromTotals, metrics.totalRequests),
    requestsByKey:
      analytics && rows
        ? new Map(settledUsage(keys, rows).map((entry) => [entry.key.id, entry.total]))
        : undefined,
  };
}

export type KeysPageInput = KeysPageModelInput & {
  presets: TimePreset[];
  preset: TimePreset;
  defaultPreset: TimePreset;
  patch: KeysSearchPatch;
  days: number;
};

export type AnalyticsPanel = {
  metrics: VerificationMetrics;
  chart: ChartState;
  series: VerificationBucket[];
  days: number;
  dimmed: boolean;
};

export type TablePanel = {
  rows: KeyRow[];
  isLoading: boolean;
  showUsage: boolean;
  dimmed: boolean;
  /** Remounts the table so its sort, page and page size follow the filters. */
  resetKey: string;
  error?: { message?: string; onRetry: () => void };
  onClearFilters?: () => void;
};

/** Omitted for a session that sees no request counts, where a range means nothing. */
export type TimePanel = {
  presets: TimePreset[];
  preset: TimePreset;
  defaultPresetId: TimePresetId;
  onPreset: (id: TimePresetId) => void;
};

export type KeysPageContent = {
  filters: FilterBarProps;
  time?: TimePanel;
  analytics?: AnalyticsPanel;
  table: TablePanel;
};

function buildDimensions(input: KeysPageInput, model: KeysPageModel): FilterDimension[] {
  const { selectedOutcomes, selectedStatus, patch, usage } = input;
  const { keys, analytics } = usage;
  const { selectedKeys } = model;

  const dimensions: FilterDimension[] = [
    {
      id: "keys",
      label: "Key",
      groupLabel: "Keys",
      options: keyOptions(keys, model.requestsByKey),
      selected: selectedKeys,
      onToggle: (value) => patch({ keys: toggle(selectedKeys, value) }),
      onClear: () => patch({ keys: undefined }),
    },
  ];

  if (analytics) {
    dimensions.push({
      id: "outcomes",
      label: "Outcome",
      groupLabel: "Outcomes",
      hideEmpty: true,
      options: outcomeOptions(model.filtered.facetCounts),
      selected: selectedOutcomes,
      onToggle: (value) =>
        patch({ outcomes: toggleFrom(OUTCOME_FILTER_KINDS, selectedOutcomes, value) }),
      onClear: () => patch({ outcomes: undefined }),
    });
  }

  dimensions.push({
    id: "status",
    label: "Status",
    groupLabel: "Statuses",
    options: statusOptions(keys),
    selected: selectedStatus,
    onToggle: (value) => patch({ status: toggleFrom(KEY_STATUSES, selectedStatus, value) }),
    onClear: () => patch({ status: undefined }),
  });

  return dimensions;
}

function buildRows(input: KeysPageInput, model: KeysPageModel): KeyRow[] {
  const { selectedOutcomes, usage } = input;
  const { analytics, byKey } = usage;

  const projected = new Map(model.filtered.keys.map((entry) => [entry.key.id, entry]));
  const rows = byKey === "unavailable" ? null : byKey.rows;

  return model.effectiveKeys.flatMap((key): KeyRow[] => {
    const matched = projected.get(key.id);
    if (matched) {
      return [{ key, usage: matched, pending: false }];
    }
    const row = rows?.get(key.id);
    // An outcome filter drops keys with no matching traffic, but never a key
    // whose own counts never arrived — those show a dash or a skeleton.
    const dropped = analytics && selectedOutcomes.length > 0 && row?.status === "ok";
    return dropped ? [] : [{ key, usage: null, pending: analytics && row?.status === "pending" }];
  });
}

export function deriveKeysPage(input: KeysPageInput): KeysPageContent {
  const dimmed =
    input.usage.aggregateState.isFetching && !input.usage.aggregateState.isInitialLoading;
  const model = deriveKeysPageModel(input);
  const { selectedOutcomes, selectedStatus, usage } = input;
  const { analytics, keysState } = usage;

  const filtering = model.selectedKeys.length + selectedOutcomes.length + selectedStatus.length > 0;

  return {
    filters: { dimensions: buildDimensions(input, model) },
    time: analytics
      ? {
          presets: input.presets,
          preset: input.preset,
          defaultPresetId: input.defaultPreset.id,
          onPreset: (id) => input.patch({ since: id === input.defaultPreset.id ? undefined : id }),
        }
      : undefined,
    analytics: analytics
      ? {
          metrics: model.metrics,
          chart: model.chart,
          series: model.filtered.totals,
          days: input.days,
          dimmed,
        }
      : undefined,
    table: {
      rows: buildRows(input, model),
      isLoading: keysState.isInitialLoading,
      showUsage: analytics,
      dimmed: analytics && dimmed,
      resetKey: JSON.stringify([
        model.selectedKeys,
        selectedOutcomes,
        selectedStatus,
        input.preset.id,
      ]),
      error: keysState.isError
        ? {
            message: keysState.error instanceof Error ? keysState.error.message : undefined,
            onRetry: keysState.refetch,
          }
        : undefined,
      onClearFilters: filtering
        ? () => input.patch({ keys: undefined, outcomes: undefined, status: undefined })
        : undefined,
    },
  };
}
