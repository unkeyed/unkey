"use client";

import { routes } from "@/lib/navigation/routes";
import { cn } from "@/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import { AgentSetup, type AgentStyle } from "./agent-setup";
import { type Mark, RowMark } from "./marks";
import type { OverviewData, UsageStat } from "./mock-data";
import type { RowVariant } from "./scenario";
import { CubeIcon, fmtCompact, fmtInt, TrendArrow, trendPct } from "./ui";

export const TEAL = "hsl(173 80% 36%)";
export const INFO = "hsl(206 100% 50%)";

export type RowItem = {
  id: string;
  title: string;
  subtitle: string;
  project?: string;
  value: string;
  spark: number[];
  errorRatio: number;
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
  const keyspaceItems: RowItem[] = data.keyspaces.map((ks) => ({
    id: ks.id,
    title: ks.name,
    subtitle: `${fmtInt(ks.keyCount)} keys`,
    project: ks.projectName,
    value: fmtCompact(ks.requests["24h"]),
    spark: ks.spark["24h"],
    errorRatio: (100 - ks.validPct) / 100,
    stroke: TEAL,
    kind: "keyspace",
    // The real API detail page, fed by the prototype tRPC interceptor.
    href: routes.apis.detail({ workspaceSlug, apiId: ks.id }),
  }));
  const ratelimitItems: RowItem[] = data.ratelimits.map((rl) => ({
    id: rl.id,
    title: rl.name,
    subtitle: `${rl.blockedPct}% blocked`,
    project: rl.projectName,
    value: fmtCompact(rl.checks["24h"]),
    spark: rl.spark["24h"],
    errorRatio: rl.blockedPct / 100,
    stroke: INFO,
    kind: "ratelimit",
    href: routes.ratelimits.detail({ workspaceSlug, namespaceId: rl.id }),
  }));

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
  children,
}: {
  title: string;
  variant: RowVariant;
  children: React.ReactNode;
}) {
  const headerBorder = variant !== "list";
  return (
    <div className="rounded-lg border border-grayA-4 bg-background">
      <div
        className={cn(
          headerBorder ? "px-4 py-3 border-b border-grayA-4" : "px-3.5 pt-3 pb-1.5",
        )}
      >
        <span className="text-[13px] font-medium text-accent-12">{title}</span>
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

function Delta({ spark }: { spark: number[] }) {
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
            errorRatio={item.errorRatio}
            stroke={item.stroke}
            className="w-20 h-9 shrink-0"
          />
        </div>
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
            errorRatio={item.errorRatio}
            stroke={item.stroke}
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
      <div className="relative shrink-0">
        <RowMark
          mark={mark}
          points={item.spark}
          errorRatio={item.errorRatio}
          stroke={item.stroke}
          className="w-40 h-7"
        />
        <div className="absolute bottom-full right-0 z-20 mb-2 hidden group-hover:block">
          <HoverStats item={item} />
        </div>
      </div>
    </Link>
  );
}

function HoverStats({ item }: { item: RowItem }) {
  const total = item.spark.reduce((acc, n) => acc + n, 0);
  const errors = Math.round(total * item.errorRatio);
  const valid = Math.max(0, total - errors);
  const isKeyspace = item.kind === "keyspace";
  return (
    <div className="w-48 rounded-lg border border-grayA-4 bg-background p-3 shadow-md">
      <div className="font-medium text-[13px] text-accent-12">
        {isKeyspace ? "Keyspace activity" : "Ratelimit activity"}
      </div>
      <div className="text-xs text-gray-9">Last 24 hours</div>
      <div className="mt-2.5 flex flex-col gap-1.5 text-xs">
        <StatRow label="Total" count={total} swatch="bg-transparent" bold />
        <StatRow label={isKeyspace ? "Valid" : "Passed"} count={valid} swatch="bg-gray-7" />
        {errors > 0 && (
          <StatRow label={isKeyspace ? "Invalid" : "Blocked"} count={errors} swatch="bg-error-9" />
        )}
      </div>
    </div>
  );
}

function StatRow({
  label,
  count,
  swatch,
  bold,
}: {
  label: string;
  count: number;
  swatch: string;
  bold?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="flex items-center gap-2 text-gray-11">
        <span className={cn("size-1.5 rounded-[1px]", swatch)} aria-hidden />
        {label}
      </span>
      <span className={cn("text-gray-12 tabular-nums", bold && "font-medium")}>
        {fmtInt(count)}
      </span>
    </div>
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

function fmtDollars(n: number): string {
  return `$${Number.isInteger(n) ? n : n.toFixed(2)}`;
}

export function UsageCard({
  usage,
  workspaceSlug,
}: {
  usage: UsageStat;
  workspaceSlug: string;
}) {
  const pct = usage.quota > 0 ? Math.min(100, (usage.billableTotal / usage.quota) * 100) : 0;
  const computePct =
    usage.computeCredits > 0
      ? Math.min(100, (usage.computeSpend / usage.computeCredits) * 100)
      : 0;
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
      <div className="mt-0.5 text-xs text-gray-9">{usage.daysLeft} days left in cycle</div>
      <div className="mt-4">
        <div className="flex justify-between text-xs">
          <span className="text-gray-11">Requests</span>
          <span className="text-accent-12 font-medium tabular-nums">
            {fmtCompact(usage.billableTotal)} / {fmtCompact(usage.quota)}
          </span>
        </div>
        <div className="mt-1.5 h-2 rounded-full bg-gray-3 overflow-hidden">
          <div className="h-full rounded-full bg-accent-12" style={{ width: `${pct}%` }} />
        </div>
      </div>
      {usage.hasComputePlan ? (
        <div className="mt-3">
          <div className="flex justify-between text-xs">
            <span className="text-gray-11">Compute</span>
            <span className="text-accent-12 font-medium tabular-nums">
              {fmtDollars(usage.computeSpend)} / {fmtDollars(usage.computeCredits)} credits
            </span>
          </div>
          <div className="mt-1.5 h-2 rounded-full bg-gray-3 overflow-hidden">
            <div className="h-full rounded-full bg-accent-12" style={{ width: `${computePct}%` }} />
          </div>
        </div>
      ) : (
        <div className="mt-3 flex items-center justify-between text-xs">
          <span className="text-gray-11">Compute</span>
          <Link
            href={routes.settings.billing({ workspaceSlug })}
            className="font-medium text-accent-11 hover:text-accent-12"
          >
            Add a plan →
          </Link>
        </div>
      )}
      <div className="mt-4 grid grid-cols-2 gap-3 text-xs">
        <div>
          <div className="text-gray-9">Verifications</div>
          <div className="text-accent-12 font-medium tabular-nums mt-0.5">
            {fmtCompact(usage.verifications)}
          </div>
        </div>
        <div>
          <div className="text-gray-9">Ratelimits</div>
          <div className="text-accent-12 font-medium tabular-nums mt-0.5">
            {fmtCompact(usage.ratelimits)}
          </div>
        </div>
      </div>
    </div>
  );
}