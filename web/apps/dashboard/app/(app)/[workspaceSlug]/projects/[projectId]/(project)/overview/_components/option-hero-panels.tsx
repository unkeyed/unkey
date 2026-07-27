"use client";

import { DeploymentStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-badge";
import type {
  KeyspaceStat,
  RatelimitStat,
  UsageStat,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import { fmtCompact, fmtInt } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/ui";
import { GithubIcon } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/ui";
import { UsageMeter } from "@/app/(app)/[workspaceSlug]/settings/billing/components/usage-meter";
import { AreaTimeseriesChart } from "@/components/charts/area-timeseries";
import type { ChartConfig } from "@/components/ui/chart";
import type { DeploymentStatus as RealDeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { routes } from "@/lib/navigation/routes";
import { cn } from "@/lib/utils";
import { CodeBranch, CodeCommit } from "@unkey/icons";
import { Button, Skeleton } from "@unkey/ui";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { CopyAgentPromptButton } from "./agent-prompt";
import type { DeploymentMock } from "./deployments-mock";
import { fmtTimeAgo } from "./deployments-mock";
import type { ActivityEvent } from "./overview-mocks";
import { projectRequestSeries } from "./overview-mocks";

const REQUEST_COLOR = "hsl(var(--activity))";
const REQUEST_FILL = "hsl(var(--info-3))";
const ERROR_COLOR = "hsl(var(--error-9))";
const ERROR_FILL = "hsl(var(--error-3))";

const TRAFFIC_CONFIG: ChartConfig = {
  valid: { label: "Requests", color: REQUEST_COLOR },
  error: { label: "Errors", color: ERROR_COLOR },
};

const TRAFFIC_FILL_COLORS = { valid: REQUEST_FILL, error: ERROR_FILL };

function LegendDot({ color, label, value }: { color: string; label: string; value: string }) {
  return (
    <span className="flex items-center gap-1.5 tabular-nums">
      <span className="size-2 shrink-0 rounded-full" style={{ backgroundColor: color }} />
      <span className="text-gray-9">{label}</span>
      <span className="font-medium text-accent-12">{value}</span>
    </span>
  );
}

// Every project's traffic panel (active or migrated) is fed the same way: an
// hourly series seeded from the project id, magnitude set by whatever
// keyspaces actually see. The chart's x-axis is anchored at render time, so it
// mount-gates behind a skeleton to avoid a server/client timestamp mismatch.
export function TrafficPanel({
  projectId,
  keyspaces,
}: { projectId: string; keyspaces: KeyspaceStat[] }) {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  const baseMagnitude = Math.max(
    1,
    keyspaces.reduce((sum, ks) => sum + ks.requests["24h"], 0) / 24,
  );
  const series = useMemo(
    () => projectRequestSeries(projectId, 24, baseMagnitude),
    [projectId, baseMagnitude],
  );

  const data = useMemo(() => {
    const nowMs = Date.now();
    return series.map((point, i) => ({
      originalTimestamp: nowMs - (series.length - 1 - i) * 3_600_000,
      valid: point.valid,
      error: point.error,
    }));
  }, [series]);

  const totalValid = series.reduce((sum, p) => sum + p.valid, 0);
  const totalError = series.reduce((sum, p) => sum + p.error, 0);

  return (
    <div className="border border-grayA-4 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-grayA-4">
        <span className="text-sm font-medium text-accent-12">Requests</span>
        <span className="text-xs text-gray-9">Last 24 hours</span>
      </div>
      <div className="flex flex-col gap-3 p-4">
        <div>
          <div className="text-2xl font-semibold text-accent-12 tabular-nums leading-tight">
            {fmtCompact(totalValid)}
          </div>
          <div className="text-[13px] text-gray-9">requests in the last 24 hours</div>
        </div>
        {mounted ? (
          <AreaTimeseriesChart
            data={data}
            config={TRAFFIC_CONFIG}
            fillColors={TRAFFIC_FILL_COLORS}
            paleFill
            height={130}
            axisFloor={0}
            formatTooltipValue={(v) => ({ value: fmtCompact(Math.round(v)) })}
            formatYTick={(v) => (v > 0 ? fmtCompact(Math.round(v)) : "")}
            hideTooltip
          />
        ) : (
          <Skeleton className="h-[130px] w-full" />
        )}
        <div className="flex items-center gap-4 text-[13px]">
          <LegendDot color={REQUEST_COLOR} label="Requests" value={fmtCompact(totalValid)} />
          <LegendDot color={ERROR_COLOR} label="Errors" value={fmtCompact(totalError)} />
        </div>
      </div>
    </div>
  );
}

function DeploymentRow({
  d,
  workspaceSlug,
  projectId,
}: {
  d: DeploymentMock;
  workspaceSlug: string;
  projectId: string;
}) {
  return (
    <div className="relative flex flex-col md:flex-row md:items-center gap-3 md:gap-4 px-4 py-3 transition-colors hover:bg-grayA-2">
      <Link
        href={routes.projects.apps.overview({ workspaceSlug, projectId, appId: d.appId })}
        className="absolute inset-0 z-10"
        aria-label={`View ${d.appName}`}
      />
      <div className="relative z-20 flex shrink-0 items-center gap-3 md:w-[168px]">
        <DeploymentStatusBadge status={d.status as RealDeploymentStatus} />
        <div className="min-w-0">
          <div className="font-mono text-[13px] font-semibold text-accent-12 truncate">
            {d.appName}
          </div>
          <div className="text-xs text-gray-9 capitalize">{d.environment}</div>
        </div>
      </div>
      <div className="relative z-20 min-w-0 flex-1">
        <div className="flex items-center gap-1.5 text-xs">
          <CodeBranch iconSize="sm-regular" className="shrink-0 text-gray-9" />
          <span className="font-mono text-accent-12 truncate">{d.branch}</span>
          <span className="shrink-0 text-gray-9">· {d.sha.slice(0, 7)}</span>
        </div>
        <div className="mt-1 flex items-center gap-1.5 text-xs text-accent-12">
          <CodeCommit iconSize="sm-regular" className="shrink-0 text-gray-9" />
          <span className="truncate">{d.message}</span>
        </div>
      </div>
      <div className="relative z-20 shrink-0 text-right text-[13px] text-gray-9">
        <div className="tabular-nums">{fmtTimeAgo(d.timeAgoMin)}</div>
        <div className="text-xs">{d.actor}</div>
      </div>
    </div>
  );
}

function NoDeploymentsBand({
  workspaceSlug,
  projectId,
}: { workspaceSlug: string; projectId: string }) {
  return (
    <div className="flex flex-col gap-4 px-4 py-6 md:flex-row md:items-center">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-gray-3 text-gray-11">
        <GithubIcon className="size-4" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="text-[13px] font-semibold text-accent-12">No deployments yet</div>
        <p className="text-[13px] text-gray-9">
          Connect a repo or hand off to your agent to deploy your app.
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <Button
          size="sm"
          variant="primary"
          render={<Link href={routes.projects.apps.new({ workspaceSlug, projectId })} />}
        >
          Connect GitHub
        </Button>
        <CopyAgentPromptButton />
      </div>
    </div>
  );
}

export function DeploymentsPanel({
  workspaceSlug,
  projectId,
  deployments,
}: {
  workspaceSlug: string;
  projectId: string;
  deployments: DeploymentMock[];
}) {
  return (
    <div className="border border-grayA-4 rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-grayA-4">
        <span className="text-sm font-medium text-accent-12">Recent deployments</span>
        {deployments.length > 0 && (
          <Link
            href={routes.projects.apps.list({ workspaceSlug, projectId })}
            className="text-xs text-gray-9 hover:text-accent-12"
          >
            View all
          </Link>
        )}
      </div>
      {deployments.length === 0 ? (
        <NoDeploymentsBand workspaceSlug={workspaceSlug} projectId={projectId} />
      ) : (
        <div className="divide-y divide-grayA-4">
          {deployments.map((d) => (
            <DeploymentRow key={d.id} d={d} workspaceSlug={workspaceSlug} projectId={projectId} />
          ))}
        </div>
      )}
    </div>
  );
}

// Compact 12-bar sparkline shared by the keyspace/ratelimit rail rows: an
// error share (top, orange) stacked over the healthy share (bottom, accent),
// a dashed baseline painted over both, and a flat tick pattern when a row has
// no traffic at all.
function MiniBars({ points, errorShare }: { points: number[]; errorShare: number }) {
  const data = points.slice(-12);
  const hasSignal = data.some((v) => v > 0);
  const H = 16;

  if (!hasSignal) {
    return (
      <div className="flex h-4 items-end gap-[2px]">
        {Array.from({ length: 12 }, (_, i) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: positional ticks
          <div key={i} className="h-0.5 w-2 bg-gray-5" />
        ))}
      </div>
    );
  }

  // Pure proportional split, no pixel floor — at rail scale a 1px error cap on
  // every bar reads as a detached dashed line (StatsListCard behaves the same:
  // small error shares are carried by the text stats, not the chart).
  const max = Math.max(...data, 1) * 1.2;
  return (
    <div className="relative inline-flex h-4 items-end gap-[2px]">
      {data.map((v, i) => {
        const total = Math.max(1, Math.round((v / max) * H));
        const errH = Math.round(errorShare * total);
        const okH = Math.max(total - errH, 0);
        return (
          // biome-ignore lint/suspicious/noArrayIndexKey: positional bars
          <div key={i} className="flex w-[3px] shrink-0 flex-col justify-end">
            <div style={{ height: `${errH}px`, backgroundColor: "hsl(var(--orange-9))" }} />
            <div style={{ height: `${okH}px`, backgroundColor: "hsl(var(--accent-4))" }} />
          </div>
        );
      })}
      <div className="pointer-events-none absolute inset-x-0 bottom-0 border-t border-dashed border-gray-5" />
    </div>
  );
}

export function KeyspacesRailCard({
  workspaceSlug,
  keyspaces,
}: {
  workspaceSlug: string;
  keyspaces: KeyspaceStat[];
}) {
  if (keyspaces.length === 0) {
    return null;
  }
  return (
    <div className="border border-grayA-4 rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-grayA-4">
        <span className="text-sm font-medium text-accent-12">Keyspaces</span>
      </div>
      <div className="divide-y divide-grayA-4">
        {keyspaces.map((ks) => {
          const errorShare = Math.max(0, Math.min(1, 1 - ks.validPct / 100));
          return (
            <Link
              key={ks.id}
              href={routes.apis.detail({ workspaceSlug, apiId: ks.id })}
              className="flex items-center justify-between gap-3 px-4 py-2.5 transition-colors hover:bg-grayA-2"
            >
              <div className="min-w-0">
                <div className="text-[13px] font-medium text-accent-12 truncate">{ks.name}</div>
                <div className="text-xs text-gray-9">{fmtInt(ks.keyCount)} keys</div>
              </div>
              <div className="flex shrink-0 items-center gap-3">
                <span className="text-[13px] tabular-nums text-gray-9">
                  {fmtCompact(ks.requests["24h"])}
                </span>
                <MiniBars points={ks.spark["24h"]} errorShare={errorShare} />
              </div>
            </Link>
          );
        })}
      </div>
    </div>
  );
}

export function RatelimitsRailCard({
  workspaceSlug,
  ratelimits,
}: {
  workspaceSlug: string;
  ratelimits: RatelimitStat[];
}) {
  if (ratelimits.length === 0) {
    return null;
  }
  return (
    <div className="border border-grayA-4 rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-grayA-4">
        <span className="text-sm font-medium text-accent-12">Ratelimits</span>
      </div>
      <div className="divide-y divide-grayA-4">
        {ratelimits.map((rl) => {
          const errorShare = Math.max(0, Math.min(1, rl.blockedPct / 100));
          return (
            <Link
              key={rl.id}
              href={routes.ratelimits.detail({ workspaceSlug, namespaceId: rl.id })}
              className="flex items-center justify-between gap-3 px-4 py-2.5 transition-colors hover:bg-grayA-2"
            >
              <div className="min-w-0">
                <div className="text-[13px] font-medium text-accent-12 truncate">{rl.name}</div>
                <div className="text-xs text-gray-9">{rl.blockedPct.toFixed(1)}% blocked</div>
              </div>
              <div className="flex shrink-0 items-center gap-3">
                <span className="text-[13px] tabular-nums text-gray-9">
                  {fmtCompact(rl.checks["24h"])}
                </span>
                <MiniBars points={rl.spark["24h"]} errorShare={errorShare} />
              </div>
            </Link>
          );
        })}
      </div>
    </div>
  );
}

export function UsageRailCard({
  workspaceSlug,
  usage,
}: {
  workspaceSlug: string;
  usage: UsageStat;
}) {
  return (
    <div className="flex flex-col gap-4 border border-grayA-4 rounded-lg p-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-accent-12">Usage</span>
        <Link
          href={routes.settings.billing({ workspaceSlug })}
          className="text-xs text-gray-9 hover:text-accent-12"
        >
          Billing
        </Link>
      </div>
      <UsageMeter
        label="Requests"
        value={`${fmtCompact(usage.billableTotal)} / ${fmtCompact(usage.quota)}`}
        fraction={usage.quota > 0 ? usage.billableTotal / usage.quota : null}
        fillClassName="bg-gray-12"
      />
      {usage.hasComputePlan ? (
        <UsageMeter
          label="Compute"
          value={`$${usage.computeSpend.toFixed(2)} / $${usage.computeCredits.toFixed(2)}`}
          fraction={usage.computeCredits > 0 ? usage.computeSpend / usage.computeCredits : null}
          fillClassName="bg-gray-12"
        />
      ) : (
        <Link
          href={routes.settings.billing({ workspaceSlug })}
          className="flex items-center justify-between text-[13px] text-gray-9 hover:text-accent-12"
        >
          <span>Compute</span>
          <span>Add a plan &rarr;</span>
        </Link>
      )}
      <div className="text-xs text-gray-9">{usage.daysLeft} days left in this cycle</div>
    </div>
  );
}

const ACTIVITY_DOT: Record<ActivityEvent["kind"], string> = {
  deploy: "bg-info-9",
  key: "bg-accent-9",
  ratelimit: "bg-orange-9",
  member: "bg-success-9",
  domain: "bg-gray-9",
};

export function ActivityRailCard({ activity }: { activity: ActivityEvent[] }) {
  if (activity.length === 0) {
    return null;
  }
  return (
    <div className="border border-grayA-4 rounded-lg overflow-hidden">
      <div className="px-4 py-3 border-b border-grayA-4">
        <span className="text-sm font-medium text-accent-12">Recent activity</span>
      </div>
      <div className="divide-y divide-grayA-4">
        {activity.map((event) => (
          <div key={event.id} className="flex items-center gap-2.5 px-4 py-2.5">
            <span className={cn("size-1.5 shrink-0 rounded-full", ACTIVITY_DOT[event.kind])} />
            <span className="min-w-0 flex-1 truncate text-xs text-gray-11">{event.text}</span>
            <span className="shrink-0 text-xs tabular-nums text-gray-9">
              {fmtTimeAgo(event.timeAgoMin)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
