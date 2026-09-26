"use client";

import { statusGroupOf } from "@/lib/collections/deploy/deployment-status";
import { trpc } from "@/lib/trpc/client";
import type { ProjectOverview } from "@/lib/trpc/routers/deploy/project/overview";

const WINDOW_MS = 7 * 24 * 60 * 60 * 1000;
const END = Math.floor(Date.now() / 60_000) * 60_000;

export const WINDOW = { startTime: END - WINDOW_MS, endTime: END };

const KS_OPTS = { trpc: { context: { skipBatch: true } } };

export function useKeyspaceTotal(keyAuthId: string): number | undefined {
  const { data } = trpc.api.overview.timeseries.useQuery(
    { keyspaceId: keyAuthId, ...WINDOW, since: "" },
    KS_OPTS,
  );
  return data?.timeseries?.reduce((a, p) => a + p.y.total, 0);
}

export type Totals = {
  keys: number;
  verified: number | undefined;
  rlRequests: number | undefined;
  rlBlockedPct: number | undefined;
  ready: number;
  failed: number;
  waiting: number;
  building: number;
};

export function useTotals(data: ProjectOverview): Totals {
  const ks = trpc.useQueries((t) =>
    data.keyspaces.map((k) =>
      t.api.overview.timeseries({ keyspaceId: k.keyAuthId, ...WINDOW, since: "" }, KS_OPTS),
    ),
  );
  const { data: rl } = trpc.ratelimit.logs.queryRatelimitTimeseriesBatch.useQuery(
    { namespaceIds: data.ratelimits.map((n) => n.id), ...WINDOW },
    { enabled: data.ratelimits.length > 0 },
  );
  const verified = ks.every((q) => q.data)
    ? ks.reduce((a, q) => a + (q.data?.timeseries?.reduce((s, p) => s + p.y.total, 0) ?? 0), 0)
    : undefined;
  let rlRequests: number | undefined;
  let rlBlockedPct: number | undefined;
  if (data.ratelimits.length === 0) {
    rlRequests = 0;
    rlBlockedPct = 0;
  } else if (rl) {
    let total = 0;
    let passed = 0;
    for (const series of Object.values(rl.timeseriesByNamespace)) {
      for (const p of series) {
        total += p.y.total;
        passed += p.y.passed;
      }
    }
    rlRequests = total;
    rlBlockedPct = total ? ((total - passed) / total) * 100 : 0;
  }
  const status = (s: string) => data.apps.filter((a) => a.latest?.status === s).length;
  return {
    keys: data.keyspaces.reduce((a, k) => a + k.keyCount, 0),
    verified,
    rlRequests,
    rlBlockedPct,
    ready: status("ready"),
    failed: status("failed"),
    waiting: status("awaiting_approval"),
    building: data.apps.filter((a) => a.latest && statusGroupOf(a.latest.status) === "building")
      .length,
  };
}
