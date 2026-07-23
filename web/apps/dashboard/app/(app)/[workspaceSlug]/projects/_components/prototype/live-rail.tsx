"use client";

import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useLiveQuery } from "@tanstack/react-db";
import { useCallback, useEffect, useMemo, useState } from "react";
import { AgentSetup, type AgentStyle } from "./agent-setup";
import type { Mark } from "./marks";
import type { UsageStat } from "./mock-data";
import { INFO, RailListShell, RailRow, RowSkeleton, type RowItem, TEAL, UsageCard } from "./rail";
import type { RowVariant } from "./scenario";
import { fmtCompact, fmtInt } from "./ui";

const DAY_MS = 24 * 60 * 60 * 1000;

export function LiveRail({
  variant,
  mark,
  agentStyle,
  workspaceSlug,
  agentDismissed,
  onDismissAgent,
}: {
  variant: RowVariant;
  mark: Mark;
  agentStyle: AgentStyle;
  workspaceSlug: string;
  agentDismissed: boolean;
  onDismissAgent: () => void;
}) {
  const isTile = variant === "tile";
  const [{ startTime, endTime }] = useState(() => {
    const end = Date.now();
    return { startTime: end - DAY_MS, endTime: end };
  });

  const keyspacesQuery = trpc.api.overview.query.useInfiniteQuery(
    { limit: 10 },
    { getNextPageParam: (last) => last.nextCursor },
  );
  const keyspaces = keyspacesQuery.data?.pages.flatMap((p) => p.apiList) ?? [];

  const nsQuery = useLiveQuery((q) => q.from({ namespace: collection.ratelimitNamespaces }));
  const namespaces = nsQuery.data ?? [];
  const nsIds = namespaces.map((n) => n.id);
  const batch = trpc.ratelimit.logs.queryRatelimitTimeseriesBatch.useQuery(
    { namespaceIds: nsIds, startTime, endTime },
    { enabled: nsIds.length > 0 },
  );

  const [keyspaceTotals, setKeyspaceTotals] = useState<Record<string, number>>({});
  const onKeyspaceTotal = useCallback((id: string, total: number) => {
    setKeyspaceTotals((prev) => (prev[id] === total ? prev : { ...prev, [id]: total }));
  }, []);

  const ratelimitItems: RowItem[] = namespaces.map((ns) => {
    const points = batch.data?.timeseriesByNamespace?.[ns.id] ?? [];
    const total = points.reduce((acc, p) => acc + (p.y?.total ?? 0), 0);
    const passed = points.reduce((acc, p) => acc + (p.y?.passed ?? 0), 0);
    const blockedPct = total > 0 ? Math.round(((total - passed) / total) * 1000) / 10 : 0;
    return {
      id: ns.id,
      title: ns.name,
      subtitle: `${blockedPct}% blocked`,
      value: fmtCompact(total),
      spark: points.map((p) => p.y?.total ?? 0),
      errorRatio: total > 0 ? (total - passed) / total : 0,
      stroke: INFO,
      kind: "ratelimit",
      href: routes.ratelimits.detail({ workspaceSlug, namespaceId: ns.id }),
    };
  });

  const { quotas } = useWorkspace();
  const usage = useMemo<UsageStat>(() => {
    const verifications = Object.values(keyspaceTotals).reduce((a, b) => a + b, 0);
    const ratelimits = ratelimitItems.reduce((acc, item) => {
      const ns = namespaces.find((n) => n.id === item.id);
      const points = batch.data?.timeseriesByNamespace?.[ns?.id ?? ""] ?? [];
      return acc + points.reduce((a, p) => a + (p.y?.total ?? 0), 0);
    }, 0);
    const now = new Date();
    const monthEnd = new Date(now.getFullYear(), now.getMonth() + 1, 1).getTime();
    return {
      billableTotal: verifications + ratelimits,
      quota: quotas?.requestsPerMonth ?? 10_000_000,
      verifications,
      ratelimits,
      daysLeft: Math.max(0, Math.ceil((monthEnd - now.getTime()) / DAY_MS)),
      hasComputePlan: false,
      computeSpend: 0,
      computeCredits: 20,
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyspaceTotals, batch.data, quotas]);

  return (
    <aside className="w-full lg:w-[320px] shrink-0 flex flex-col gap-4">
      {!agentDismissed && <AgentSetup style={agentStyle} onDismiss={onDismissAgent} />}

      <RailListShell title="Keyspaces" variant={variant}>
        {keyspacesQuery.isLoading ? (
          <>
            <RowSkeleton isTile={isTile} />
            <RowSkeleton isTile={isTile} />
            <RowSkeleton isTile={isTile} />
          </>
        ) : keyspaces.length === 0 ? (
          <EmptyRow label="No keyspaces yet" />
        ) : (
          keyspaces.map((ks) => (
            <LiveKeyspaceRow
              key={ks.id}
              keyspace={ks}
              variant={variant}
              mark={mark}
              isTile={isTile}
              workspaceSlug={workspaceSlug}
              startTime={startTime}
              endTime={endTime}
              onTotal={onKeyspaceTotal}
            />
          ))
        )}
      </RailListShell>

      {namespaces.length > 0 && (
        <RailListShell title="Ratelimits" variant={variant}>
          {batch.isLoading ? (
            <>
              <RowSkeleton isTile={isTile} />
              <RowSkeleton isTile={isTile} />
            </>
          ) : (
            ratelimitItems.map((item) => (
              <RailRow key={item.id} item={item} variant={variant} mark={mark} />
            ))
          )}
        </RailListShell>
      )}

      <UsageCard usage={usage} workspaceSlug={workspaceSlug} />
    </aside>
  );
}

function LiveKeyspaceRow({
  keyspace,
  variant,
  mark,
  isTile,
  workspaceSlug,
  startTime,
  endTime,
  onTotal,
}: {
  keyspace: { id: string; name: string; keyspaceId: string | null; keyCount: number };
  variant: RowVariant;
  mark: Mark;
  isTile: boolean;
  workspaceSlug: string;
  startTime: number;
  endTime: number;
  onTotal: (id: string, total: number) => void;
}) {
  const query = trpc.api.overview.timeseries.useQuery(
    { startTime, endTime, since: "", keyspaceId: keyspace.keyspaceId ?? "" },
    { enabled: Boolean(keyspace.keyspaceId) },
  );

  const points = query.data?.timeseries ?? [];
  const total = points.reduce((acc, p) => acc + (p.y?.total ?? 0), 0);
  const valid = points.reduce((acc, p) => acc + (p.y?.valid ?? 0), 0);
  const validPct = total > 0 ? Math.round((valid / total) * 1000) / 10 : 0;

  useEffect(() => {
    if (query.data) {
      onTotal(keyspace.id, total);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.data, total]);

  if (query.isLoading) {
    return <RowSkeleton isTile={isTile} />;
  }

  const item: RowItem = {
    id: keyspace.id,
    title: keyspace.name,
    subtitle: `${fmtInt(keyspace.keyCount)} keys · ${validPct}% valid`,
    value: fmtCompact(total),
    spark: points.map((p) => p.y?.total ?? 0),
    errorRatio: total > 0 ? (total - valid) / total : 0,
    stroke: TEAL,
    kind: "keyspace",
    href: routes.apis.detail({ workspaceSlug, apiId: keyspace.id }),
  };
  return <RailRow item={item} variant={variant} mark={mark} />;
}

function EmptyRow({ label }: { label: string }) {
  return <div className="px-3.5 py-6 text-center text-xs text-gray-9">{label}</div>;
}
