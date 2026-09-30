import { describe, expect, it } from "vitest";
import { emptyOutcomes } from "~/components/analytics/analytics-transform";
import { type KeyUsage, applyFilters, keyUsage, sumSeries } from "~/components/analytics/key-usage";
import type { VerificationBucket } from "~/components/analytics/schema/analytics.schema";
import { defaultTimePreset } from "~/components/analytics/time-presets";
import type { Key } from "~/components/keys-table/schema/keys.schema";
import type { KeyUsageRow, KeysUsage } from "~/hooks/use-keys-usage";
import {
  type KeysPageInput,
  type KeysPageModelInput,
  deriveKeysPage,
  deriveKeysPageModel,
} from "./keys-page-model";

function key(id: string, name: string, enabled = true): Key {
  return { id, name, start: `${id}_pre`, createdAt: 0, expires: null, enabled };
}

function bucket(
  time: number,
  valid: number,
  outcomes: Partial<VerificationBucket> = {},
): VerificationBucket {
  const full = { ...emptyOutcomes(), ...outcomes };
  const error =
    full.rateLimited +
    full.expired +
    full.disabled +
    full.usageExceeded +
    full.insufficientPermissions +
    full.forbidden;
  return { ...full, time, valid, error, other: 0, total: valid + error };
}

const prod = keyUsage(key("k1", "Production"), [
  bucket(1, 10, { rateLimited: 2 }),
  bucket(2, 5, { expired: 1 }),
]);
const dev = keyUsage(key("k2", "Development"), [bucket(1, 3), bucket(2, 0, { forbidden: 4 })]);
const idle = keyUsage(key("k3", "Idle"), [bucket(1, 0), bucket(2, 0)]);
const aggregate = [bucket(1, 13, { rateLimited: 2 }), bucket(2, 5, { expired: 1, forbidden: 4 })];
const keys = [prod, dev, idle];

describe("applyFilters", () => {
  it("returns the given series and every key when no outcome is selected", () => {
    const result = applyFilters(keys, aggregate, []);

    expect(result.totals).toBe(aggregate);
    expect(result.keys.map((k) => k.key.id)).toEqual(["k1", "k2", "k3"]);
    expect(result.facetCounts.valid).toBe(18);
    expect(result.facetCounts.forbidden).toBe(4);
  });

  it("projects buckets onto the selected outcomes and drops emptied keys", () => {
    const result = applyFilters(keys, aggregate, ["forbidden"]);

    expect(result.keys.map((k) => k.key.id)).toEqual(["k2"]);
    expect(result.keys[0].total).toBe(4);
    expect(result.keys[0].valid).toBe(0);
    expect(result.totals.map((b) => b.total)).toEqual([0, 4]);
  });

  it("keeps valid traffic only when the valid outcome is selected", () => {
    const result = applyFilters(keys, aggregate, ["valid", "rateLimited"]);

    expect(result.keys.map((k) => k.key.id)).toEqual(["k1", "k2"]);
    expect(result.totals.map((b) => b.total)).toEqual([15, 5]);
    expect(result.totals[0].expired).toBe(0);
  });

  it("counts facets over the given series, whatever the outcome filter", () => {
    const result = applyFilters([prod], sumSeries([prod]), ["forbidden"]);

    expect(result.facetCounts.valid).toBe(15);
    expect(result.facetCounts.rateLimited).toBe(2);
    expect(result.keys).toEqual([]);
  });
});

describe("sumSeries", () => {
  it("adds the keys' buckets together, in time order", () => {
    const totals = sumSeries([dev, prod]);

    expect(totals.map((b) => b.time)).toEqual([1, 2]);
    expect(totals.map((b) => b.total)).toEqual([15, 10]);
    expect(totals[1].forbidden).toBe(4);
    expect(totals[1].expired).toBe(1);
  });

  it("is an empty series for no keys", () => {
    expect(sumSeries([])).toEqual([]);
  });
});

function ok(usage: KeyUsage): KeyUsageRow {
  return { key: usage.key, status: "ok", usage };
}

const settled = [ok(prod), ok(dev), ok(idle)];

function byKeyOf(rows: KeyUsageRow[], isPending = false): KeysUsage["byKey"] {
  return {
    rows: new Map(rows.map((row) => [row.key.id, row])),
    isPending,
    retryFailed: () => {},
  };
}

function usageOf(overrides: Partial<KeysUsage> = {}): KeysUsage {
  return {
    keys: [prod.key, dev.key, idle.key],
    keysState: { isInitialLoading: false, isError: false, error: null, refetch: () => {} },
    aggregate,
    aggregateState: {
      isInitialLoading: false,
      isFetching: false,
      isError: false,
      error: null,
      refetch: () => {},
    },
    analytics: true,
    byKey: byKeyOf(settled),
    ...overrides,
  };
}

function model(overrides: Partial<KeysPageModelInput> = {}) {
  return deriveKeysPageModel({
    usage: usageOf(),
    selectedKeys: [],
    selectedOutcomes: [],
    selectedStatus: [],
    ...overrides,
  });
}

describe("deriveKeysPageModel", () => {
  it("charts the account series when nothing is filtered", () => {
    const result = model();

    expect(result.filtered.totals).toBe(aggregate);
    expect(result.metrics.totalRequests).toBe(25);
    expect(result.chart.kind).toBe("populated");
    expect(result.missingFromTotals).toBe(false);
    expect(result.requestsByKey?.get("k1")).toBe(18);
    expect(result.requestsByKey?.get("k3")).toBe(0);
  });

  it("sums the selected keys' own series once a key filter narrows the page", () => {
    const result = model({ selectedKeys: ["k1"] });

    expect(result.filtered.totals.map((b) => b.total)).toEqual([12, 6]);
    expect(result.filtered.keys.map((k) => k.key.id)).toEqual(["k1"]);
    expect(result.metrics.totalRequests).toBe(18);
    expect(result.chart.kind).toBe("populated");
  });

  it("sums only the keys a status filter leaves in play", () => {
    const retired = key("k4", "Retired", false);
    const result = model({
      usage: usageOf({
        keys: [prod.key, retired],
        byKey: byKeyOf([ok(prod), ok(keyUsage(retired, [bucket(1, 7)]))]),
      }),
      selectedStatus: ["disabled"],
    });

    expect(result.effectiveKeys.map((k) => k.id)).toEqual(["k4"]);
    expect(result.metrics.totalRequests).toBe(7);
    expect(result.chart.kind).toBe("populated");
  });

  it("reports an error when a selected key's usage never arrived", () => {
    const result = model({
      usage: usageOf({ byKey: byKeyOf([{ key: prod.key, status: "error" }, ok(dev), ok(idle)]) }),
      selectedKeys: ["k1"],
    });

    expect(result.missingFromTotals).toBe(true);
    expect(result.chart).toMatchObject({
      kind: "error",
      message: "Couldn't load usage for some of these keys",
    });
  });

  it("ignores a failed key the selection excludes", () => {
    const result = model({
      usage: usageOf({ byKey: byKeyOf([ok(prod), { key: dev.key, status: "error" }, ok(idle)]) }),
      selectedKeys: ["k1"],
    });

    expect(result.missingFromTotals).toBe(false);
    expect(result.chart.kind).toBe("populated");
  });

  it("projects the account series onto an outcome filter alone, ignoring the per-key rows", () => {
    const result = model({
      usage: usageOf({ byKey: byKeyOf([ok(prod), { key: dev.key, status: "error" }, ok(idle)]) }),
      selectedOutcomes: ["forbidden"],
    });

    expect(result.filtered.totals.map((b) => b.total)).toEqual([0, 4]);
    expect(result.metrics.totalRequests).toBe(4);
    expect(result.filtered.facetCounts.valid).toBe(18);
    expect(result.missingFromTotals).toBe(false);
    expect(result.chart.kind).toBe("populated");
  });

  it("drops a key selection the key list could not load", () => {
    const result = model({
      usage: usageOf({
        keysState: { isInitialLoading: false, isError: true, error: null, refetch: () => {} },
      }),
      selectedKeys: ["k1"],
    });

    expect(result.selectedKeys).toEqual([]);
    expect(result.filtered.totals).toBe(aggregate);
    expect(result.metrics.totalRequests).toBe(25);
    expect(result.missingFromTotals).toBe(false);
  });

  it("waits on the per-key requests only while a filter narrows the page", () => {
    const pending = usageOf({ byKey: byKeyOf(settled, true) });

    expect(model({ usage: pending }).chart.kind).toBe("populated");
    expect(model({ usage: pending, selectedKeys: ["k1"] }).chart.kind).toBe("loading");
  });

  it("charts the account series without per-key usage, and errors once a key filter needs it", () => {
    const noFanOut = usageOf({ byKey: "unavailable" });

    expect(model({ usage: noFanOut }).chart.kind).toBe("populated");
    expect(model({ usage: noFanOut }).filtered.keys).toEqual([]);
    expect(model({ usage: noFanOut, selectedKeys: ["k1"] }).chart).toEqual({
      kind: "error",
      message: "Usage per key isn't available for this account.",
    });
  });

  it("counts no requests per key without analytics", () => {
    const result = model({ usage: usageOf({ analytics: false, byKey: "unavailable" }) });

    expect(result.requestsByKey).toBeUndefined();
    expect(result.filtered.totals).toBe(aggregate);
  });
});

const preset = defaultTimePreset(0);

function page(overrides: Partial<KeysPageInput> = {}) {
  return deriveKeysPage({
    usage: usageOf(),
    selectedKeys: [],
    selectedOutcomes: [],
    selectedStatus: [],
    presets: [preset],
    preset,
    defaultPreset: preset,
    patch: () => {},
    days: 7,
    ...overrides,
  });
}

describe("deriveKeysPage", () => {
  it("lists every key with its usage when nothing is filtered", () => {
    const { table } = page();

    expect(table.rows.map((row) => row.key.id)).toEqual(["k1", "k2", "k3"]);
    expect(table.rows.every((row) => row.usage !== null)).toBe(true);
    expect(table.error).toBeUndefined();
    expect(table.onClearFilters).toBeUndefined();
  });

  it("drops keys with no traffic matching the outcome filter", () => {
    const { table } = page({ selectedOutcomes: ["forbidden"] });

    expect(table.rows.map((row) => row.key.id)).toEqual(["k2"]);
    expect(table.onClearFilters).toBeDefined();
  });

  it("keeps a key the outcome filter cannot judge, pending or failed", () => {
    const pending = page({
      usage: usageOf({
        byKey: byKeyOf([{ key: prod.key, status: "pending" }, ok(dev), ok(idle)], true),
      }),
      selectedOutcomes: ["forbidden"],
    });
    const failed = page({
      usage: usageOf({ byKey: byKeyOf([{ key: prod.key, status: "error" }, ok(dev), ok(idle)]) }),
      selectedOutcomes: ["forbidden"],
    });

    expect(pending.table.rows.map((row) => row.key.id)).toEqual(["k1", "k2"]);
    expect(pending.table.rows[0]).toMatchObject({ usage: null, pending: true });
    expect(failed.table.rows.map((row) => row.key.id)).toEqual(["k1", "k2"]);
    expect(failed.table.rows[0]).toMatchObject({ usage: null, pending: false });
  });

  it("keeps every key with no usage at all when the per-key numbers are unavailable", () => {
    const { table, analytics } = page({
      usage: usageOf({ byKey: "unavailable" }),
      selectedOutcomes: ["forbidden"],
    });

    expect(table.rows.map((row) => row.key.id)).toEqual(["k1", "k2", "k3"]);
    expect(table.rows.every((row) => row.usage === null && !row.pending)).toBe(true);
    expect(analytics?.chart.kind).toBe("populated");
  });

  it("counts no key as pending without analytics", () => {
    const { table, analytics } = page({
      usage: usageOf({ analytics: false, byKey: "unavailable" }),
    });

    expect(table.rows.map((row) => row.pending)).toEqual([false, false, false]);
    expect(table.showUsage).toBe(false);
    expect(analytics).toBeUndefined();
  });

  it("filters the rows by status", () => {
    const retired = key("k4", "Retired", false);
    const { table } = page({
      usage: usageOf({ keys: [prod.key, retired], byKey: byKeyOf([ok(prod)]) }),
      selectedStatus: ["disabled"],
    });

    expect(table.rows.map((row) => row.key.id)).toEqual(["k4"]);
  });

  it("empties the selection when the key list could not load", () => {
    const { table } = page({
      usage: usageOf({
        keysState: {
          isInitialLoading: false,
          isError: true,
          error: new Error("nope"),
          refetch: () => {},
        },
      }),
      selectedKeys: ["k1"],
    });

    expect(table.rows.map((row) => row.key.id)).toEqual(["k1", "k2", "k3"]);
    expect(table.error).toMatchObject({ message: "nope" });
  });

  it("offers a retry for the failed per-key requests unless the account series failed", () => {
    const failedRows = page({
      usage: usageOf({ byKey: byKeyOf([{ key: prod.key, status: "error" }, ok(dev), ok(idle)]) }),
      selectedKeys: ["k1"],
    });
    const failedAggregate = page({
      usage: usageOf({
        aggregateState: {
          isInitialLoading: false,
          isFetching: false,
          isError: true,
          error: new Error("That time range isn't available. Try a shorter range."),
          refetch: () => {},
        },
      }),
    });

    expect(failedRows.analytics?.chart).toMatchObject({
      kind: "error",
      message: "Couldn't load usage for some of these keys",
    });
    expect(failedAggregate.analytics?.chart).toEqual({
      kind: "error",
      message: "That time range isn't available. Try a shorter range.",
    });
  });

  it("remounts the table whenever a filter or the time range changes", () => {
    expect(page().table.resetKey).not.toBe(page({ selectedKeys: ["k1"] }).table.resetKey);
    expect(page().table.resetKey).not.toBe(page({ selectedStatus: ["expired"] }).table.resetKey);
    expect(page().table.resetKey).not.toBe(page({ selectedOutcomes: ["valid"] }).table.resetKey);
  });

  it("offers the outcome dimension only with analytics", () => {
    const noAnalytics = page({ usage: usageOf({ analytics: false, byKey: "unavailable" }) });

    expect(page().filters.dimensions.map((d) => d.id)).toEqual(["keys", "outcomes", "status"]);
    expect(noAnalytics.filters.dimensions.map((d) => d.id)).toEqual(["keys", "status"]);
    expect(noAnalytics.time).toBeUndefined();
  });
});
