import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";
import { computeMetrics } from "~/components/analytics/analytics-transform";
import { formatCount } from "~/components/analytics/format";
import { HeaderStat } from "~/components/analytics/header-stat";
import { RangeControl } from "~/components/analytics/range-control";
import { availableAnalyticsPeriods } from "~/components/analytics/schema/analytics.schema";
import {
  REJECTED_COLOR,
  VALID_COLOR,
  VerificationsChart,
} from "~/components/analytics/verifications-chart";
import { PortalFooter } from "~/components/portal-footer";
import { PortalHeader } from "~/components/portal-header";
import { type KeyUsage, fakeKeyBreakdown } from "./-key-breakdown-data";
import { Picker } from "./-picker";

const LAYOUTS = ["Attached", "Separate", "Side by side"] as const;
const PERIODS = availableAnalyticsPeriods(30);

const searchSchema = z.object({
  v: z.number().int().min(1).max(LAYOUTS.length).catch(1),
  range: z.union([z.literal(1), z.literal(7), z.literal(30)]).catch(7),
});

export const Route = createFileRoute("/prototypes/key-breakdown")({
  validateSearch: searchSchema,
  component: KeyBreakdownPrototype,
});

function KeyBreakdownPrototype() {
  const { v, range } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });
  const [chrome, setChrome] = useState(true);
  const layout = LAYOUTS[v - 1];

  const { totals, keys } = useMemo(() => fakeKeyBreakdown(range), [range]);
  const metrics = useMemo(() => computeMetrics(totals), [totals]);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.metaKey && e.key === ".") {
        e.preventDefault();
        setChrome((value) => !value);
      }
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  const setRange = (days: number) =>
    navigate({ search: (prev) => ({ ...prev, range: days as 1 | 7 | 30 }), replace: true });
  navigate({ search: (prev) => ({ ...prev, key: undefined }), replace: true });

  const stats = (
    <div className="grid divide-y divide-primary/10 border-primary/10 border-b sm:grid-cols-3 sm:divide-x sm:divide-y-0">
      <HeaderStat label="Total requests" value={formatCount(metrics.totalRequests)} />
      <HeaderStat label="Valid" value={formatCount(metrics.validRequests)} swatch={VALID_COLOR} />
      <HeaderStat
        label="Invalid"
        value={formatCount(metrics.errorRequests)}
        swatch={REJECTED_COLOR}
      />
    </div>
  );

  const chart = (
    <div className="p-4" style={{ height: 312 }}>
      <VerificationsChart buckets={totals} days={range} />
    </div>
  );

  const grandTotal = totals.reduce((a, b) => a + b.total, 0);
  const list = <KeyList keys={keys} grandTotal={grandTotal} />;

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <PortalHeader
        scopes={["keys:read", "analytics:read"]}
        appName="Acme Ltd"
        returnUrl="#return"
      />
      <main
        className={`mx-auto w-full flex-1 px-4 pt-8 pb-12 sm:px-8 ${layout === "Side by side" ? "max-w-6xl" : "max-w-5xl"}`}
      >
        <header className="mb-6 flex flex-col items-start gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
          <div className="flex flex-col gap-1">
            <h1 className="font-semibold text-gray-12 text-xl">Analytics</h1>
            <p className="text-gray-11 text-sm">
              Monitor verification activity and usage trends for your API keys.
            </p>
          </div>
          <RangeControl options={PERIODS} value={range} onChange={setRange} />
        </header>

        {layout === "Attached" && (
          <section className="rounded-lg border border-primary/10 bg-background">
            {stats}
            {chart}
            <div className="border-primary/10 border-t">
              <ListHeading count={keys.length} inset />
              {list}
            </div>
          </section>
        )}

        {layout === "Separate" && (
          <div className="flex flex-col gap-6">
            <section className="rounded-lg border border-primary/10 bg-background">
              {stats}
              {chart}
            </section>
            <section>
              <ListHeading count={keys.length} />
              <div className="rounded-lg border border-primary/10 bg-background">{list}</div>
            </section>
          </div>
        )}

        {layout === "Side by side" && (
          <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
            <section className="rounded-lg border border-primary/10 bg-background">
              {stats}
              {chart}
            </section>
            <section className="flex flex-col rounded-lg border border-primary/10 bg-background">
              <ListHeading count={keys.length} inset compact />
              <KeyList keys={keys} grandTotal={grandTotal} compact />
            </section>
          </div>
        )}
      </main>
      <PortalFooter />
      {chrome && (
        <Picker
          label="layout"
          names={[...LAYOUTS]}
          index={v - 1}
          onChange={(i) => navigate({ search: (prev) => ({ ...prev, v: i + 1 }), replace: true })}
          bottom={24}
          keyboard
        />
      )}
    </div>
  );
}

function ListHeading({
  count,
  inset,
  compact,
}: { count: number; inset?: boolean; compact?: boolean }) {
  return (
    <div
      className={`flex items-baseline justify-between ${inset ? "px-5 pt-4 pb-1 sm:px-6" : "mb-3"} ${compact ? "border-primary/10 border-b pb-3" : ""}`}
    >
      <h2 className="font-medium text-gray-12 text-sm">By key</h2>
      <span className="text-gray-10 text-xs">{count} keys</span>
    </div>
  );
}

function KeyList({
  keys,
  grandTotal,
  compact,
}: {
  keys: KeyUsage[];
  grandTotal: number;
  compact?: boolean;
}) {
  return (
    <ul className="divide-y divide-primary/10">
      {keys.map((k) => {
        return (
          <li key={k.id}>
            <div
              className={`grid w-full items-center gap-4 px-5 py-3 text-left sm:px-6 ${
                compact
                  ? "grid-cols-[minmax(0,1fr)_auto]"
                  : "grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1.6fr)]"
              }`}
            >
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="truncate font-medium text-gray-12 text-sm">{k.name}</span>
                  {!k.enabled && <span className="text-gray-9 text-xs">Disabled</span>}
                </div>
                <span className="block truncate font-mono text-gray-10 text-xs">{k.start}••••</span>
              </div>
              {!compact && (
                <div className="flex flex-col">
                  <span className="font-medium text-gray-12 text-sm tabular-nums">
                    {formatCount(k.total)}
                  </span>
                  <span className="text-gray-10 text-xs tabular-nums">
                    {grandTotal ? `${((k.total / grandTotal) * 100).toFixed(0)}% · ` : ""}
                    {formatCount(k.error)} invalid
                  </span>
                </div>
              )}
              <div className={`flex flex-col items-end gap-1 ${compact ? "" : "sm:items-stretch"}`}>
                {compact && (
                  <span className="font-medium text-gray-12 text-sm tabular-nums">
                    {formatCount(k.total)}
                  </span>
                )}
                <Sparkline buckets={k.buckets} width={compact ? 96 : undefined} />
              </div>
            </div>
          </li>
        );
      })}
    </ul>
  );
}

function Sparkline({ buckets, width }: { buckets: KeyUsage["buckets"]; width?: number }) {
  const max = Math.max(...buckets.map((b) => b.total), 1);
  if (buckets.every((b) => b.total === 0)) {
    return (
      <div
        className="flex h-7 items-center justify-center rounded-sm bg-gray-2 text-[11px] text-gray-10"
        style={{ width }}
      >
        No usage
      </div>
    );
  }
  return (
    <div
      className="flex h-7 items-end gap-px rounded-sm bg-gray-2 px-1"
      style={{ width }}
      aria-hidden="true"
    >
      {buckets.map((b) => {
        const h = Math.max(Math.round((b.total / max) * 24), b.total > 0 ? 1 : 0);
        const e =
          b.total > 0 ? Math.max(Math.round((b.error / b.total) * h), b.error > 0 ? 1 : 0) : 0;
        return (
          <div key={b.time} className="flex min-w-0 flex-1 flex-col justify-end">
            <div className="w-full bg-error-9" style={{ height: e }} />
            <div className="w-full bg-gray-7" style={{ height: h - e }} />
          </div>
        );
      })}
    </div>
  );
}
