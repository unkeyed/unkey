"use client";

import { formatNumber } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { cn } from "@/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import { AgentSetup, type AgentStyle } from "./agent-setup";
import { type Bucket, type Mark, RowMark, type SeriesLabels } from "./marks";
import type { OverviewData, UsageStat } from "./mock-data";
import type { RowVariant } from "./scenario";
import { keyspaceSeries, ratelimitSeries } from "./series";
import { CubeIcon, TrendArrow, fmtCompact, fmtInt, trendPct } from "./ui";

// Series colours per data type, so a keyspace row and a ratelimit row are
// distinguishable and both follow the chart scheme.
const KEYSPACE_OK = "hsl(var(--chart-verify-ok))";
const KEYSPACE_BAD = "hsl(var(--chart-verify-bad))";
const RATELIMIT_OK = "hsl(var(--chart-limit-ok))";
const RATELIMIT_BAD = "hsl(var(--chart-limit-bad))";

export type RowItem = {
  id: string;
  title: string;
  subtitle: string;
  project?: string;
  value: string;
  spark: number[];
  /** Hourly valid/error pairs behind `spark`, when the caller has them. */
  buckets?: Bucket[];
  errorRatio: number;
  /** Colour for the error share; pairs with `stroke`. */
  errorStroke?: string;
  /** Series names for the chart's hover readout. */
  labels?: SeriesLabels;
  stroke: string;
  kind: "keyspace" | "ratelimit";
  href: Route;
};

export const STUB_HREF = "#" as Route;

export function Rail({
  data,
  variant,
  mark,
  agentStyle,
  workspaceSlug,
  agentDismissed,
  onDismissAgent,
}: {
  data: OverviewData;
  variant: RowVariant;
  mark: Mark;
  agentStyle: AgentStyle;
  workspaceSlug: string;
  agentDismissed: boolean;
  onDismissAgent: () => void;
}) {
  // TODO: cap both lists at the 3 most active (by 24h volume) with a "View all"
  // footer, the same treatment the overview's apps card uses. Applies to the
  // project overview's resource cards too — see ResourceLists in
  // projects/[projectId]/(project)/overview/_components/project-overview.tsx.
  //
  // Same generated series the project overview draws, so a keyspace's chart and
  // hover numbers match wherever you see it.
  const keyspaceItems: RowItem[] = data.keyspaces.map((ks) => {
    const series = keyspaceSeries(ks);
    return {
      id: ks.id,
      title: ks.name,
      subtitle: `${fmtInt(ks.keyCount)} keys`,
      project: ks.projectName,
      value: fmtCompact(series.total),
      spark: series.totals,
      buckets: series.buckets,
      errorRatio: (100 - ks.validPct) / 100,
      stroke: KEYSPACE_OK,
      errorStroke: KEYSPACE_BAD,
      labels: { ok: "valid", bad: "invalid" },
      kind: "keyspace",
      // The real API detail page, fed by the prototype tRPC interceptor.
      href: routes.apis.detail({ workspaceSlug, apiId: ks.id }),
    };
  });
  const ratelimitItems: RowItem[] = data.ratelimits.map((rl) => {
    const series = ratelimitSeries(rl);
    return {
      id: rl.id,
      title: rl.name,
      subtitle: `${rl.blockedPct}% blocked`,
      project: rl.projectName,
      value: fmtCompact(series.total),
      spark: series.totals,
      buckets: series.buckets,
      errorRatio: rl.blockedPct / 100,
      stroke: RATELIMIT_OK,
      errorStroke: RATELIMIT_BAD,
      labels: { ok: "passed", bad: "blocked" },
      kind: "ratelimit",
      href: routes.ratelimits.detail({ workspaceSlug, namespaceId: rl.id }),
    };
  });

  return (
    <aside className="w-full lg:w-[320px] shrink-0 flex flex-col gap-4">
      {!agentDismissed && <AgentSetup style={agentStyle} onDismiss={onDismissAgent} />}
      {keyspaceItems.length > 0 && (
        <RailListShell title="Keyspaces" variant={variant}>
          {keyspaceItems.map((item) => (
            <RailRow key={item.id} item={item} variant={variant} mark={mark} />
          ))}
        </RailListShell>
      )}
      {ratelimitItems.length > 0 && (
        <RailListShell title="Ratelimits" variant={variant}>
          {ratelimitItems.map((item) => (
            <RailRow key={item.id} item={item} variant={variant} mark={mark} />
          ))}
        </RailListShell>
      )}
      <UsageCard usage={data.usage} workspaceSlug={workspaceSlug} />
    </aside>
  );
}

export function RailListShell({
  title,
  variant,
  subtitle,
  count,
  viewAllHref,
  children,
}: {
  title: string;
  variant: RowVariant;
  /** States what the rows' numbers mean and over what window. */
  subtitle?: string;
  /** How many rows the list has, shown as a pill beside the title. */
  count?: number;
  viewAllHref?: Route;
  children: React.ReactNode;
}) {
  const headerBorder = variant !== "list";
  return (
    <div className="rounded-lg border border-grayA-4 bg-background">
      <div
        className={cn(
          "flex items-center justify-between gap-2",
          headerBorder ? "px-4 py-3 border-b border-grayA-4" : "px-3.5 pt-3 pb-1.5",
        )}
      >
        <span className="flex min-w-0 items-center gap-2">
          <span className="text-[13px] font-medium text-accent-12">{title}</span>
          {count !== undefined && (
            <span className="rounded-full bg-grayA-3 px-1.5 py-0.5 text-[11px] font-medium tabular-nums text-gray-11">
              {count}
            </span>
          )}
          {subtitle && <span className="truncate text-xs text-gray-9">{subtitle}</span>}
        </span>
        {viewAllHref && (
          <Link
            href={viewAllHref}
            className="text-xs text-gray-9 hover:text-accent-12 hover:underline"
          >
            View all
          </Link>
        )}
      </div>
      <div
        className={
          variant === "tile"
            ? "p-3 flex flex-col gap-2"
            : variant === "flat"
              ? "p-1.5 flex flex-col gap-0.5"
              : "divide-y divide-grayA-4"
        }
      >
        {children}
      </div>
    </div>
  );
}

function RowSubtitle({ item, className }: { item: RowItem; className?: string }) {
  return (
    <div className={cn("flex items-center gap-1 text-xs text-gray-9 min-w-0", className)}>
      <span className="truncate">{item.subtitle}</span>
      {item.project && (
        <span className="inline-flex items-center gap-1 shrink-0">
          <span className="text-gray-7">·</span>
          <CubeIcon className="size-3" />
          <span className="truncate max-w-[96px]">{item.project}</span>
        </span>
      )}
    </div>
  );
}

export function Delta({ spark }: { spark: number[] }) {
  const pct = trendPct(spark);
  const up = pct >= 0;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 text-[11px] tabular-nums shrink-0",
        up ? "text-success-11" : "text-gray-11",
      )}
    >
      <TrendArrow up={up} className="size-2.5" />
      {Math.abs(pct)}%
    </span>
  );
}

export function RailRow({
  item,
  variant,
  mark,
}: {
  item: RowItem;
  variant: RowVariant;
  mark: Mark;
}) {
  if (variant === "detailed") {
    return (
      <Link
        href={item.href}
        className="group flex items-center gap-3 px-3.5 py-3 hover:bg-grayA-2 transition-colors"
      >
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span className="text-[13px] text-accent-12 truncate leading-4">{item.title}</span>
            <span className="text-[13px] font-medium text-accent-12 tabular-nums shrink-0">
              {item.value}
            </span>
          </div>
          <div className="mt-1 flex items-center justify-between gap-2">
            <RowSubtitle item={item} className="flex-1" />
            <Delta spark={item.spark} />
          </div>
        </div>
      </Link>
    );
  }

  if (variant === "tile") {
    return (
      <Link
        href={item.href}
        className="group flex flex-col gap-3 rounded-lg border border-grayA-4 bg-gray-1 p-3 hover:border-grayA-7 transition-colors"
      >
        <div className="flex items-center justify-between gap-2">
          <span className="text-[13px] text-accent-12 truncate">{item.title}</span>
          <Delta spark={item.spark} />
        </div>
        <div className="flex items-end justify-between gap-3">
          <div className="min-w-0">
            <div className="text-lg font-semibold text-accent-12 tabular-nums leading-none">
              {item.value}
            </div>
            <RowSubtitle item={item} className="mt-1" />
          </div>
          <RowMark
            mark={mark}
            points={item.spark}
            buckets={item.buckets}
            errorRatio={item.errorRatio}
            stroke={item.stroke}
            errorStroke={item.errorStroke}
            labels={item.labels}
            className="w-20 h-9 shrink-0"
          />
        </div>
      </Link>
    );
  }

  // "metric" — Vercel's observability row: the name reads as the label, the
  // value sits under it with its unit spelled out, and the chart takes the rest
  // of the width. No delta: a bare percentage next to a bare number invites the
  // question "since when?" and answers neither.
  if (variant === "metric") {
    return (
      <Link
        href={item.href}
        className="group flex items-center gap-4 px-4 py-3.5 transition-colors hover:bg-grayA-2"
      >
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-baseline gap-1.5">
            <span className="truncate text-[13px] text-gray-11">{item.title}</span>
            <span className="shrink-0 text-xs text-gray-9">· {item.subtitle}</span>
          </div>
          <div className="mt-1 text-[15px] font-semibold leading-none tabular-nums text-accent-12">
            {item.value}
          </div>
        </div>
        <RowMark
          mark={mark}
          points={item.spark}
          buckets={item.buckets}
          errorRatio={item.errorRatio}
          stroke={item.stroke}
          errorStroke={item.errorStroke}
          labels={item.labels}
          className="h-8 w-[152px] shrink-0"
          fill
        />
      </Link>
    );
  }

  // "hybrid" — big metric + a contained chart stacked on the right (no overlap).
  if (variant === "hybrid") {
    return (
      <Link
        href={item.href}
        className="group flex items-center justify-between gap-3 px-3.5 py-3 transition-colors hover:bg-grayA-2"
      >
        <div className="min-w-0 flex-1">
          <div className="truncate text-[13px] font-medium text-accent-12">{item.title}</div>
          <RowSubtitle item={item} className="mt-0.5" />
        </div>
        <div className="flex shrink-0 flex-col items-end gap-1">
          <div className="flex items-center gap-1.5">
            <span className="text-[15px] font-semibold leading-none tabular-nums text-accent-12">
              {item.value}
            </span>
            <Delta spark={item.spark} />
          </div>
          <RowMark
            mark="bars"
            points={item.spark}
            buckets={item.buckets}
            errorRatio={item.errorRatio}
            stroke={item.stroke}
            errorStroke={item.errorStroke}
            labels={item.labels}
            className="h-6"
          />
        </div>
      </Link>
    );
  }

  // "graph" | "flat"
  const isFlat = variant === "flat";
  return (
    <Link
      href={item.href}
      className={cn(
        "group relative flex items-center gap-3 transition-colors hover:bg-grayA-2",
        isFlat ? "rounded-md px-1.5 py-2" : "px-3.5 py-2.5",
      )}
    >
      <div className="min-w-0 flex-1">
        <div className="text-[13px] text-accent-12 truncate leading-4">{item.title}</div>
        <RowSubtitle item={item} className="mt-0.5" />
      </div>
      <RowMark
        mark={mark}
        points={item.spark}
        buckets={item.buckets}
        errorRatio={item.errorRatio}
        stroke={item.stroke}
        errorStroke={item.errorStroke}
        labels={item.labels}
        className="w-40 h-7 shrink-0"
      />
    </Link>
  );
}

export function RowSkeleton({ isTile }: { isTile: boolean }) {
  if (isTile) {
    return <div className="h-[76px] rounded-lg border border-grayA-4 bg-gray-2 animate-pulse" />;
  }
  return (
    <div className="flex items-center gap-3 px-3.5 py-2.5">
      <div className="size-7 rounded-md bg-gray-3 animate-pulse shrink-0" />
      <div className="flex-1 space-y-1.5">
        <div className="h-3 w-24 rounded bg-gray-3 animate-pulse" />
        <div className="h-2.5 w-32 rounded bg-gray-2 animate-pulse" />
      </div>
    </div>
  );
}

function usd(n: number): string {
  return `$${n.toFixed(2)}`;
}

function usageFillClass(fraction: number): string {
  if (fraction >= 1) {
    return "bg-error-9";
  }
  if (fraction >= 0.8) {
    return "bg-warning-9";
  }
  return "bg-accent-12";
}

function UsageMeterRow({
  label,
  value,
  fraction,
}: {
  label: string;
  value: string;
  fraction: number | null;
}) {
  const pct = fraction === null ? 0 : Math.min(100, Math.max(0, fraction * 100));
  return (
    <div className="flex w-full flex-col gap-1.5">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-[13px] text-gray-11">{label}</span>
        <span className="text-[13px] font-medium text-accent-12 tabular-nums">{value}</span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-grayA-3">
        {fraction !== null && (
          <div
            className={cn("h-full rounded-full", usageFillClass(fraction))}
            style={{ width: `${pct}%` }}
          />
        )}
      </div>
    </div>
  );
}

// "Ledger" treatment (picked 2026-07-29): the two products as two labelled
// meters, using the billing page's own vocabulary — "Valid key verifications
// and ratelimits" / formatNumber pair for API management, "Usage this period"
// / "$X of $Y credits" for Compute. Never "requests", never activity metrics.
export function UsageCard({
  usage,
  workspaceSlug,
}: {
  usage: UsageStat;
  workspaceSlug: string;
}) {
  const fraction = usage.quota > 0 ? usage.billableTotal / usage.quota : 0;
  const pct = Math.round(fraction * 100);
  return (
    <div className="rounded-lg border border-grayA-4 bg-background p-4">
      <div className="flex items-center justify-between">
        <span className="text-[13px] font-medium text-accent-12">Usage</span>
        <Link
          href={routes.settings.billing({ workspaceSlug })}
          className="text-xs text-gray-9 hover:text-accent-12"
        >
          Billing
        </Link>
      </div>
      <div className="mt-4">
        <div className="text-[11px] font-medium uppercase tracking-wide text-gray-9">
          API management
        </div>
        <div className="mt-2">
          <UsageMeterRow
            label="Verifications & ratelimits"
            value={`${formatNumber(usage.billableTotal)} / ${formatNumber(usage.quota)} (${pct}%)`}
            fraction={fraction}
          />
        </div>
      </div>
      <div className="my-4 border-t border-grayA-3" />
      <div>
        <div className="text-[11px] font-medium uppercase tracking-wide text-gray-9">Compute</div>
        <div className="mt-2">
          {usage.hasComputePlan ? (
            <UsageMeterRow
              label="Usage this period"
              value={`${usd(usage.computeSpend)} of ${usd(usage.computeCredits)} credits`}
              fraction={usage.computeCredits > 0 ? usage.computeSpend / usage.computeCredits : null}
            />
          ) : (
            <div className="flex items-baseline justify-between gap-3">
              <span className="text-[13px] text-gray-11">No active plan</span>
              <Link
                href={routes.settings.billing({ workspaceSlug })}
                className="text-[13px] font-medium text-accent-11 hover:text-accent-12"
              >
                Choose a plan →
              </Link>
            </div>
          )}
        </div>
      </div>
      <p className="mt-4 text-xs text-gray-9">{usage.daysLeft} days left in cycle</p>
    </div>
  );
}
