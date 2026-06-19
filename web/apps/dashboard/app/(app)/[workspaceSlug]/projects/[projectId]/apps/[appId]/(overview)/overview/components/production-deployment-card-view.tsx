"use client";

import type { ChartConfig } from "@/components/ui/chart";
import { githubUrl } from "@/lib/github-url";
import type { FlagCode } from "@/lib/trpc/routers/deploy/network/utils";
import type { LastExit } from "@/lib/types/deploy";
import { cn } from "@/lib/utils";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  CodeBranch,
  CodeCommit,
  Layers3,
  Plus,
  TriangleWarning2,
} from "@unkey/icons";
import { Badge, Button, TimestampInfo } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { useMemo, useState } from "react";
import { MetadataCell } from "../../../components/active-deployment-card/components/metadata-cell";
import { DottedLink } from "../../../components/dotted-link";
import { Avatar } from "../../../components/git-avatar";
import { RegionFlag } from "../../../components/region-flag";
import { Card } from "../../components/card";
import {
  type AreaChartPoint,
  AreaTimeseriesChart,
} from "../../deployments/[deploymentId]/network/unkey-flow/components/overlay/node-details-panel/components/chart/area-timeseries-chart";
import { type Pulse, formatCount, formatRps, formatStamp } from "./g-pulse";

const BLUE = "hsl(var(--info-9))";
const BLUE_FILL = "hsl(var(--info-3))";
const ERROR = "hsl(var(--error-9))";

const COMPACT_CONFIG: ChartConfig = {
  total: { label: "Requests", color: BLUE },
  errors: { label: "Errors", color: ERROR },
};

// Stable references — passing inline literals here would change identity on
// every hover-driven re-render and reset the chart's internal cursor state.
const FILL_COLORS = { total: BLUE_FILL };

export const formatRpsTooltip = (value: number) => ({ value: value.toFixed(1), unit: "req/s" });

// RPS y-ticks — the chart's default tick formatter renders bytes ("KiB"), wrong
// for request counts.
function formatRpsYTick(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  return v >= 1000 ? `${(v / 1000).toFixed(1)}k` : `${Math.round(v)}`;
}

export type DeploymentDisplayStatus = "live" | "deploying" | "crashing" | "failed" | "stopped";

export type ProductionDeploymentViewModel = {
  status: DeploymentDisplayStatus;
  rolledBack: boolean;
  // The deployment that was live before the rollback (status "superseded").
  // Only populated when rolledBack; the now-live half of the pair is this vm.
  rolledBackFrom: { commitSha: string | null; commitMessage: string | null } | null;
  branch: string | null;
  commitSha: string | null;
  commitMessage: string | null;
  image: string | null;
  repoFullName: string | null;
  forkRepositoryFullName: string | null;
  authorHandle: string | null;
  authorAvatarUrl: string | null;
  createdAt: number;
  primaryDomain: { hostname: string; url: string } | null;
  additionalDomainCount: number;
  // Set only when the app has no custom domain (just the generated *.unkey.app)
  // — links to the settings page where the custom-domain form lives.
  addCustomDomainHref?: string;
  canRollback: boolean;
  regions: { id: string; name: string; flagCode: FlagCode }[];
  runningCount: number;
  targetCount: number;
  cpuMillicores: number;
  memoryMib: number;
  storageMib: number;
  lastExit: LastExit | null;
  logsHref?: string;
  requestsHref?: string;
  // Set only for crashing/failed — a context action to diagnose (crash logs /
  // build error). Label + href differ by which state.
  diagnostic?: { label: string; href: string };
};

const STATUS_META: Record<
  DeploymentDisplayStatus,
  { label: string; dotClass: string; style?: string }
> = {
  live: { label: "Live", dotClass: "bg-success-9" },
  deploying: { label: "Deploying", dotClass: "", style: "hsl(38, 92%, 50%)" },
  crashing: { label: "Crashing", dotClass: "bg-error-9" },
  failed: { label: "Failed", dotClass: "bg-error-9" },
  stopped: { label: "Stopped", dotClass: "bg-gray-9" },
};

function StatusDot({ status }: { status: DeploymentDisplayStatus }) {
  const meta = STATUS_META[status];
  return (
    <span
      className={cn("size-2 shrink-0 rounded-full", meta.dotClass)}
      style={meta.style ? { backgroundColor: meta.style } : undefined}
    />
  );
}

function DomainHero({ vm }: { vm: ProductionDeploymentViewModel }) {
  return (
    <div className="flex items-center gap-2 min-w-0">
      {vm.primaryDomain ? (
        <a
          href={vm.primaryDomain.url}
          target="_blank"
          rel="noopener noreferrer"
          className="font-mono tracking-tight text-base font-semibold text-accent-12 truncate hover:underline decoration-dashed underline-offset-3"
        >
          {vm.primaryDomain.hostname}
        </a>
      ) : (
        <span className="font-mono text-base font-semibold text-gray-9 truncate">
          No domain yet
        </span>
      )}
      {vm.additionalDomainCount > 0 && (
        <span className="rounded-full px-1.5 py-0.5 bg-grayA-3 text-gray-12 text-[11px] leading-[18px] font-mono tabular-nums shrink-0">
          +{vm.additionalDomainCount}
        </span>
      )}
      {vm.addCustomDomainHref && (
        <Button variant="outline" size="sm" asChild className="shrink-0 border-dashed">
          <Link href={vm.addCustomDomainHref as Route}>
            <Plus iconSize="sm-regular" />
            Add custom domain
          </Link>
        </Button>
      )}
    </div>
  );
}

function LegendStat({
  color,
  label,
  value,
  alert,
}: {
  color: string;
  label: string;
  value: string;
  alert?: boolean;
}) {
  return (
    <span className="flex items-center gap-1.5 whitespace-nowrap tabular-nums">
      <span className="size-2 shrink-0 rounded-full" style={{ backgroundColor: color }} />
      <span className="text-gray-9">{label}</span>
      <span className={cn("font-medium", alert ? "text-error-11" : "text-accent-12")}>{value}</span>
    </span>
  );
}

function GitHubLink({ href, children }: { href: string | undefined; children: React.ReactNode }) {
  if (!href) {
    return <>{children}</>;
  }
  return (
    <DottedLink href={href} external>
      {children}
    </DottedLink>
  );
}

function ProductionCardHeader({
  vm,
  onRollback,
  actionsSlot,
}: {
  vm: ProductionDeploymentViewModel;
  onRollback?: () => void;
  actionsSlot?: React.ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-3 border-b border-gray-4">
      <div className="flex items-center gap-2 min-w-0">
        <DomainHero vm={vm} />
      </div>
      <div className="flex items-center gap-2 shrink-0">
        {vm.diagnostic && (
          <Button variant="outline" size="sm" asChild className="border-errorA-4 text-error-11">
            <Link href={vm.diagnostic.href as Route}>
              <TriangleWarning2 iconSize="sm-regular" />
              {vm.diagnostic.label}
            </Link>
          </Button>
        )}
        {vm.logsHref && (
          <Button variant="outline" size="sm" asChild>
            <Link href={vm.logsHref as Route}>
              <Layers3 iconSize="sm-regular" />
              Logs
            </Link>
          </Button>
        )}
        {vm.requestsHref && (
          <Button variant="outline" size="sm" asChild>
            <Link href={vm.requestsHref as Route}>
              <ArrowOppositeDirectionY iconSize="sm-regular" />
              Requests
            </Link>
          </Button>
        )}
        {/* Stays "Instant Rollback" even while rolled back, so you can roll
            back again to a different version (Vercel keeps it in the header).
            The undo action lives in the rollback banner instead. */}
        {(vm.canRollback || vm.rolledBack) && (
          <Button variant="outline" size="sm" onClick={onRollback}>
            <ArrowDottedRotateAnticlockwise iconSize="sm-regular" />
            Instant Rollback
          </Button>
        )}
        {actionsSlot}
      </div>
    </div>
  );
}

// Slim rolled-back notice that tucks UNDER the top of the card (rounded top,
// square bottom, negative bottom margin so the card overlaps its lower edge).
// One line: the paused-deploys notice + Undo. The from→to detail and state live
// in the Source and Status cells, so this stays a thin strip rather than a block.
function RollbackBanner({ onUndoRollback }: { onUndoRollback?: () => void }) {
  return (
    <div className="relative z-0 -mb-3 flex items-center justify-between gap-3 rounded-t-[14px] border border-b-0 border-warning-6 bg-warning-3 px-4 pt-2.5 pb-5">
      <div className="flex items-center gap-1.5 text-[13px] min-w-0">
        <ArrowDottedRotateAnticlockwise
          iconSize="sm-regular"
          className="text-warning-11 shrink-0"
        />
        <span className="font-medium text-accent-12 shrink-0">Rolled back</span>
        <span className="text-gray-12 truncate">
          — new production deploys are paused until you undo.
        </span>
      </div>
      {onUndoRollback && (
        <Button variant="outline" size="sm" onClick={onUndoRollback} className="shrink-0 bg-gray-1">
          Undo Rollback
        </Button>
      )}
    </div>
  );
}

function MetaGrid({ vm }: { vm: ProductionDeploymentViewModel }) {
  const sourceRepo = vm.forkRepositoryFullName || vm.repoFullName;
  const cpuVcpu = vm.cpuMillicores / 1000;
  const statusMeta = STATUS_META[vm.status];
  return (
    <div className="grid grid-cols-2 gap-y-4 gap-x-6 items-start">
      <MetadataCell label="Status">
        {vm.rolledBack ? (
          <span className="flex items-center gap-2 text-[13px] text-accent-12">
            <StatusDot status="live" />
            Live
            <Badge variant="warning" size="sm">
              Rolled back
            </Badge>
          </span>
        ) : (
          <span className="flex items-center gap-2 text-[13px] text-accent-12">
            <StatusDot status={vm.status} />
            {statusMeta.label}
          </span>
        )}
      </MetadataCell>

      <MetadataCell label="Region">
        {vm.regions.length > 0 ? (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
            {vm.regions.map((r) => (
              <span key={r.id} className="flex items-center gap-1.5 text-[13px] text-accent-12">
                <RegionFlag flagCode={r.flagCode} size="xs" shape="square" />
                {r.name}
              </span>
            ))}
          </div>
        ) : (
          <span className="text-gray-9 text-[13px]">—</span>
        )}
      </MetadataCell>

      <MetadataCell label="Resources">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-[13px] text-gray-9">
          <span>
            <span className="text-accent-12 tabular-nums">{cpuVcpu}</span> vCPU
          </span>
          <span aria-hidden>·</span>
          <span>
            <span className="text-accent-12 tabular-nums">{vm.memoryMib}</span> MiB
          </span>
        </div>
      </MetadataCell>

      <MetadataCell label="Instances">
        <span className="text-[13px] text-gray-9">
          <span className="text-accent-12 tabular-nums">{vm.runningCount}</span> running
        </span>
      </MetadataCell>

      <MetadataCell label="Source">
        <div className="flex flex-col gap-1 min-w-0">
          {vm.branch && (
            <GitHubLink href={githubUrl.branch(sourceRepo, vm.branch)}>
              <span className="flex items-center gap-1.5">
                <CodeBranch iconSize="sm-regular" className="text-accent-12 shrink-0" />
                <span className="font-mono text-[13px] text-accent-12 truncate max-w-40">
                  {vm.branch}
                </span>
              </span>
            </GitHubLink>
          )}
          {vm.commitSha && (
            <div className="flex items-center gap-1.5 min-w-0">
              <GitHubLink href={githubUrl.commit(sourceRepo, vm.commitSha)}>
                <span className="flex items-center gap-1.5">
                  <CodeCommit iconSize="sm-regular" className="text-accent-12 shrink-0" />
                  <span className="font-mono text-[13px] text-accent-12">
                    {vm.commitSha.slice(0, 7)}
                  </span>
                </span>
              </GitHubLink>
              {vm.commitMessage && (
                <span className="text-[13px] text-accent-12 truncate min-w-0">
                  {vm.commitMessage}
                </span>
              )}
            </div>
          )}
          {vm.rolledBack && vm.rolledBackFrom && (
            <div className="flex items-center gap-1.5 min-w-0 text-gray-9">
              <CodeCommit iconSize="sm-regular" className="text-gray-9 shrink-0" />
              <span className="font-mono text-[13px] line-through shrink-0">
                {vm.rolledBackFrom.commitSha ? vm.rolledBackFrom.commitSha.slice(0, 7) : "—"}
              </span>
              {vm.rolledBackFrom.commitMessage && (
                <span className="text-[13px] line-through truncate min-w-0">
                  {vm.rolledBackFrom.commitMessage}
                </span>
              )}
            </div>
          )}
          {!vm.branch && !vm.commitSha && (
            <span className="font-mono text-[13px] text-accent-12 truncate">{vm.image ?? "—"}</span>
          )}
        </div>
      </MetadataCell>

      <MetadataCell label="Created">
        <div className="flex items-center gap-2">
          <Avatar src={vm.authorAvatarUrl} alt="Author" />
          {vm.authorHandle && (
            <span className="font-medium text-accent-12 text-[13px] truncate">
              {vm.authorHandle}
            </span>
          )}
          <TimestampInfo
            value={vm.createdAt}
            displayType="relative"
            className="text-gray-9 text-[13px] shrink-0"
          />
        </div>
      </MetadataCell>
    </div>
  );
}

// Two-column Option-G card: the domain is the hero (header), a cumulative
// "requests this <window>" headline + heartbeat chart on the left, deployment
// metadata on the right. The chart's hovered datum drives the legend values and
// the UTC timestamp (PlanetScale pattern) instead of a floating tooltip.
export function ProductionDeploymentCardView({
  vm,
  pulse,
  onRollback,
  onUndoRollback,
  actionsSlot,
}: {
  vm: ProductionDeploymentViewModel;
  pulse: Pulse;
  onRollback?: () => void;
  onUndoRollback?: () => void;
  actionsSlot?: React.ReactNode;
}) {
  const [active, setActive] = useState<AreaChartPoint | null>(null);

  const reqValue = active ? Number(active.total) || 0 : pulse.rpsCurrent;
  const errValue = active ? Number(active.errors) || 0 : pulse.errorsCurrent;
  const stampTs = active ? active.originalTimestamp : pulse.latestTimestamp;
  const xDomain = useMemo<[number, number] | undefined>(
    () =>
      pulse.series.length > 1
        ? [
            pulse.series[0].originalTimestamp,
            pulse.series[pulse.series.length - 1].originalTimestamp,
          ]
        : undefined,
    [pulse.series],
  );

  return (
    <div className="relative">
      {vm.rolledBack && <RollbackBanner onUndoRollback={onUndoRollback} />}
      <Card className="relative z-10 bg-base-12 flex flex-col">
        <ProductionCardHeader vm={vm} onRollback={onRollback} actionsSlot={actionsSlot} />
        <div className="grid grid-cols-1 md:grid-cols-2">
          <div className="flex flex-col gap-2 p-4 md:border-r border-gray-4">
            <div className="flex items-start justify-between gap-2">
              <div className="flex flex-col">
                <span className="text-2xl font-semibold text-accent-12 tabular-nums leading-tight">
                  {formatCount(pulse.cumulative)}
                </span>
                <span className="text-[13px] text-gray-9">requests {pulse.windowLabel}</span>
              </div>
              <span className="text-[13px] tabular-nums text-gray-9">
                {formatStamp(stampTs, pulse.windowKey, active !== null)}
              </span>
            </div>
            <AreaTimeseriesChart
              data={pulse.series}
              config={COMPACT_CONFIG}
              fillColors={FILL_COLORS}
              paleFill
              height={120}
              axisFloor={0}
              formatTooltipValue={formatRpsTooltip}
              formatYTick={formatRpsYTick}
              xAxisDomain={xDomain}
              xAxisUTC
              hideTooltip
              onActiveChange={setActive}
            />
            <div className="flex items-center gap-4 text-[13px]">
              <LegendStat color={BLUE} label="Requests" value={formatRps(reqValue)} />
              <LegendStat
                color={ERROR}
                label="Errors"
                value={formatRps(errValue)}
                alert={errValue > 0}
              />
            </div>
          </div>
          <div className="p-4">
            <MetaGrid vm={vm} />
          </div>
        </div>
      </Card>
    </div>
  );
}
