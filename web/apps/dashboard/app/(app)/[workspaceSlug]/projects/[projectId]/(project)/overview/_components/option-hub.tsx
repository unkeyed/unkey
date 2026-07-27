"use client";

import { DeploymentStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-badge";
import {
  GaugeIcon,
  GithubIcon,
  KeyIcon,
  TerminalIcon,
  fmtCompact,
  fmtInt,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/ui";
import { UsageMeter } from "@/app/(app)/[workspaceSlug]/settings/billing/components/usage-meter";
import { type AreaChartPoint, AreaTimeseriesChart } from "@/components/charts/area-timeseries";
import type { ChartConfig } from "@/components/ui/chart";
import { routes } from "@/lib/navigation/routes";
import { CodeBranch, CodeCommit, Cube } from "@unkey/icons";
import { Button, Skeleton } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { type ComponentType, useEffect, useState } from "react";
import { CopyAgentPromptButton } from "./agent-prompt";
import { type DeploymentMock, fmtTimeAgo } from "./deployments-mock";
import type { OverviewProjectData } from "./overview-data";
import { projectRequestSeries } from "./overview-mocks";

type ProjectData = OverviewProjectData["project"];
type AppData = ProjectData["apps"][number];

const REQUEST_STROKE = "hsl(var(--activity))";
const REQUEST_FILL = "hsl(var(--info-3))";
const ERROR_STROKE = "hsl(var(--error-9))";
const ERROR_FILL = "hsl(var(--error-3))";

const TRAFFIC_CHART_CONFIG: ChartConfig = {
  valid: { label: "Requests", color: REQUEST_STROKE },
  error: { label: "Errors", color: ERROR_STROKE },
};
const TRAFFIC_FILL_COLORS = { valid: REQUEST_FILL, error: ERROR_FILL };

// Deterministic 24-120 spread per app id so sibling apps' charts don't all
// share the same magnitude — pure string hash, no shared RNG state needed.
function seedMagnitude(id: string): number {
  let h = 0;
  for (let i = 0; i < id.length; i++) {
    h = (h * 31 + id.charCodeAt(i)) | 0;
  }
  return 24 + (Math.abs(h) % 96);
}

function useMounted(): boolean {
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);
  return mounted;
}

// Dave's own sticky-note idea, off a Zuplo screenshot: a center app node with
// keyspaces/environments/keys as connected side-cards. Rebuilt here as a
// hierarchy — apps are the loud, living things up top; keyspaces and
// ratelimits are resources visually subordinate but rich, "connected to" them.
export function OptionHub({ data }: { data: OverviewProjectData }) {
  const { scenario, project, keyspaces, ratelimits, usage, deployments, workspaceSlug } = data;
  const mounted = useMounted();
  const deployByApp = new Map(deployments.map((d) => [d.appId, d] as const));

  const counts = [
    project.apps.length > 0 ? pluralize(project.apps.length, "app") : null,
    keyspaces.length > 0 ? pluralize(keyspaces.length, "keyspace") : null,
    ratelimits.length > 0 ? pluralize(ratelimits.length, "ratelimit") : null,
  ].filter((c): c is string => c !== null);

  const showConnectedFull = scenario !== "new";
  const showFootBand = scenario !== "new";
  const showDeployHero = project.apps.length === 0 && scenario === "new";

  return (
    <div className="flex flex-col gap-5">
      <Header
        project={project}
        workspaceSlug={workspaceSlug}
        counts={counts}
        hideDeploy={showDeployHero}
      />

      {project.apps.length > 0 ? (
        <div className="flex flex-col gap-4">
          {project.apps.map((app) => (
            <AppCard
              key={app.id}
              app={app}
              project={project}
              workspaceSlug={workspaceSlug}
              deployment={deployByApp.get(app.id)}
              mounted={mounted}
            />
          ))}
        </div>
      ) : scenario === "new" ? (
        <DeployHeroCard workspaceSlug={workspaceSlug} projectId={project.id} />
      ) : (
        <DeployCompactCard workspaceSlug={workspaceSlug} projectId={project.id} />
      )}

      {showConnectedFull ? (
        <ConnectedResources
          keyspaces={keyspaces}
          ratelimits={ratelimits}
          workspaceSlug={workspaceSlug}
          projectId={project.id}
        />
      ) : (
        <ConnectedResourcesPlaceholders workspaceSlug={workspaceSlug} projectId={project.id} />
      )}

      {showFootBand && <FootBand usage={usage} workspaceSlug={workspaceSlug} />}
    </div>
  );
}

function pluralize(count: number, unit: string): string {
  return `${count} ${unit}${count === 1 ? "" : "s"}`;
}

function Header({
  project,
  workspaceSlug,
  counts,
  hideDeploy,
}: {
  project: ProjectData;
  workspaceSlug: string;
  counts: string[];
  hideDeploy: boolean;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="min-w-0">
        <h1 className="truncate text-[22px] font-semibold tracking-tight leading-tight text-accent-12">
          {project.name}
        </h1>
        <div className="mt-1.5 flex flex-wrap items-center gap-x-1.5 text-xs text-gray-9">
          <span className="font-mono">{project.id}</span>
          {counts.map((c) => (
            <span key={c} className="flex items-center gap-1.5">
              <span aria-hidden>·</span>
              {c}
            </span>
          ))}
        </div>
      </div>
      {!hideDeploy && (
        <Button
          size="md"
          variant="primary"
          render={
            <Link href={routes.projects.apps.new({ workspaceSlug, projectId: project.id })} />
          }
        >
          Deploy
        </Button>
      )}
    </div>
  );
}

function AppCard({
  app,
  project,
  workspaceSlug,
  deployment,
  mounted,
}: {
  app: AppData;
  project: ProjectData;
  workspaceSlug: string;
  deployment: DeploymentMock | undefined;
  mounted: boolean;
}) {
  const href = routes.projects.apps.overview({
    workspaceSlug,
    projectId: project.id,
    appId: app.id,
  });
  const domain = `${app.name}-${project.name}.unkey.app`;

  const series = projectRequestSeries(app.id, 24, seedMagnitude(app.id));
  const total24h = series.reduce((sum, p) => sum + p.valid + p.error, 0);
  const now = mounted ? Date.now() : 0;
  const hourMs = 60 * 60 * 1000;
  const chartData: AreaChartPoint[] = series.map((p, i) => ({
    originalTimestamp: now - (series.length - 1 - i) * hourMs,
    valid: p.valid,
    error: p.error,
  }));

  return (
    <div className="group relative flex flex-col gap-4 rounded-lg border border-gray-4 bg-background p-4 hover:border-grayA-7 sm:flex-row sm:items-center sm:justify-between">
      <Link href={href} className="absolute inset-0 z-10" aria-label={`Open ${app.name}`} />

      <div className="relative z-0 flex min-w-0 items-start gap-3">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-gray-3 text-gray-11">
          {app.source === "github" ? (
            <GithubIcon className="size-4" />
          ) : (
            <TerminalIcon className="size-4" />
          )}
        </span>
        <div className="min-w-0">
          <div className="truncate text-sm font-medium text-accent-12">{app.name}</div>
          <div className="truncate font-mono text-xs text-gray-9">{domain}</div>
          <div className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-[13px]">
            {deployment ? (
              <>
                <DeploymentStatusBadge status={deployment.status} />
                <span className="flex shrink-0 items-center gap-1 font-mono text-accent-12">
                  <CodeBranch iconSize="sm-regular" className="text-gray-9" />
                  {deployment.branch}
                </span>
                <span className="flex shrink-0 items-center gap-1 font-mono text-accent-12">
                  <CodeCommit iconSize="sm-regular" className="text-gray-9" />
                  {deployment.sha.slice(0, 7)}
                </span>
                <span className="min-w-0 flex-1 truncate text-gray-9">{deployment.message}</span>
                <span className="shrink-0 text-gray-9">{fmtTimeAgo(deployment.timeAgoMin)}</span>
              </>
            ) : (
              <span className="text-gray-9">Not deployed yet</span>
            )}
          </div>
        </div>
      </div>

      <div className="relative z-0 flex shrink-0 flex-col items-end gap-1">
        <div className="flex flex-col items-end">
          <span className="text-lg font-semibold tabular-nums text-accent-12">
            {fmtCompact(total24h)}
          </span>
          <span className="text-xs text-gray-9">requests · 24h</span>
        </div>
        {mounted ? (
          <div className="w-[220px]">
            <AreaTimeseriesChart
              data={chartData}
              config={TRAFFIC_CHART_CONFIG}
              fillColors={TRAFFIC_FILL_COLORS}
              paleFill
              hideAxes
              hideTooltip
              axisFloor={0}
              height={64}
            />
          </div>
        ) : (
          <Skeleton className="h-16 w-[220px] rounded-md" />
        )}
      </div>
    </div>
  );
}

function DeployCompactCard({
  workspaceSlug,
  projectId,
}: {
  workspaceSlug: string;
  projectId: string;
}) {
  return (
    <div className="flex flex-col items-start justify-between gap-4 rounded-lg border border-gray-4 bg-background p-4 sm:flex-row sm:items-center">
      <div className="min-w-0">
        <div className="text-[15px] font-semibold text-accent-12">No apps yet</div>
        <p className="mt-0.5 text-[13px] text-gray-9">
          Connect a repo or hand off to your agent — deploy your app in minutes.
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

function DeployHeroCard({
  workspaceSlug,
  projectId,
}: {
  workspaceSlug: string;
  projectId: string;
}) {
  return (
    <div className="flex flex-col items-center gap-4 rounded-lg border border-gray-4 bg-background py-14 text-center">
      <span className="flex size-12 items-center justify-center rounded-lg bg-gray-3 text-gray-11">
        <Cube className="size-6" />
      </span>
      <div>
        <h2 className="text-[17px] font-semibold text-accent-12">Deploy your first app</h2>
        <p className="mx-auto mt-1 max-w-sm text-[13px] text-gray-9">
          Connect a Git repository for automatic deployments, or hand this project to your coding
          agent and let it ship the first version.
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Button
          size="md"
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

type ResourceRow = {
  id: string;
  href: Route;
  icon: ComponentType<{ className?: string }>;
  name: string;
  sub: string;
  compact: string;
  bars: { success: number; error: number }[];
};

// Error share expressed as [0, 1] of each hourly bucket, so the mini bars
// read as "mostly healthy, small error sliver" rather than two independent
// series — keyspaces only carry an overall validPct, ratelimits a blockedPct.
function splitBars(spark: number[], errorShare: number): { success: number; error: number }[] {
  const share = Math.min(1, Math.max(0, errorShare));
  return spark.map((v) => {
    const error = Math.round(v * share);
    return { success: Math.max(0, v - error), error };
  });
}

function ConnectedResources({
  keyspaces,
  ratelimits,
  workspaceSlug,
  projectId,
}: {
  keyspaces: OverviewProjectData["keyspaces"];
  ratelimits: OverviewProjectData["ratelimits"];
  workspaceSlug: string;
  projectId: string;
}) {
  const keyspaceRows: ResourceRow[] = keyspaces.map((ks) => ({
    id: ks.id,
    href: routes.apis.detail({ workspaceSlug, apiId: ks.id }),
    icon: KeyIcon,
    name: ks.name,
    sub: `${fmtInt(ks.keyCount)} keys`,
    compact: fmtCompact(ks.requests["24h"]),
    bars: splitBars(ks.spark["24h"], (100 - ks.validPct) / 100),
  }));

  const ratelimitRows: ResourceRow[] = ratelimits.map((rl) => ({
    id: rl.id,
    href: routes.ratelimits.detail({ workspaceSlug, namespaceId: rl.id }),
    icon: GaugeIcon,
    name: rl.name,
    sub: `${rl.blockedPct.toFixed(1)}% blocked`,
    compact: fmtCompact(rl.checks["24h"]),
    bars: splitBars(rl.spark["24h"], rl.blockedPct / 100),
  }));

  return (
    <div className="ml-4 border-l border-grayA-4 pl-5">
      <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
        <ResourceListCard
          title="Keyspaces"
          viewAllHref={routes.projects.keyspaces({ workspaceSlug, projectId })}
          rows={keyspaceRows}
          emptyText="No keyspaces yet — create one to issue API keys."
        />
        <ResourceListCard
          title="Ratelimits"
          viewAllHref={routes.projects.ratelimits({ workspaceSlug, projectId })}
          rows={ratelimitRows}
          emptyText="No ratelimits yet — add one to protect an endpoint."
        />
      </div>
    </div>
  );
}

function ResourceListCard({
  title,
  viewAllHref,
  rows,
  emptyText,
}: {
  title: string;
  viewAllHref: Route;
  rows: ResourceRow[];
  emptyText: string;
}) {
  return (
    <div className="rounded-lg border border-gray-4 bg-background">
      <div className="flex items-center justify-between gap-4 border-b border-grayA-4 px-4 py-3">
        <span className="text-[13px] font-medium text-accent-12">{title}</span>
        {rows.length > 0 && (
          <Link href={viewAllHref} className="text-xs text-gray-9 hover:text-accent-12">
            View all
          </Link>
        )}
      </div>
      {rows.length === 0 ? (
        <div className="px-4 py-8 text-center">
          <p className="text-[13px] text-gray-9">{emptyText}</p>
          <Link
            href={viewAllHref}
            className="mt-2 inline-block text-xs font-medium text-accent-11 hover:text-accent-12"
          >
            Go to {title.toLowerCase()} →
          </Link>
        </div>
      ) : (
        <div className="divide-y divide-grayA-4">
          {rows.map((row) => (
            <Link
              key={row.id}
              href={row.href}
              className="flex items-center gap-3 px-4 py-3 hover:bg-grayA-2"
            >
              <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-gray-3 text-gray-11">
                <row.icon className="size-3.5" />
              </span>
              <div className="min-w-0 flex-1">
                <div className="truncate text-[13px] text-accent-12">{row.name}</div>
                <div className="truncate text-xs text-gray-9">{row.sub}</div>
              </div>
              <div className="flex shrink-0 items-center gap-3">
                <span className="text-[13px] tabular-nums text-accent-12">{row.compact}</span>
                <MiniStackedBars bars={row.bars} />
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

function MiniStackedBars({ bars }: { bars: { success: number; error: number }[] }) {
  const max = Math.max(1, ...bars.map((b) => b.success + b.error));
  const isEmpty = bars.every((b) => b.success + b.error === 0);

  return (
    <div className="relative flex h-8 w-16 items-end gap-[3px]">
      {isEmpty
        ? bars.map((_, i) => (
            <span
              // biome-ignore lint/suspicious/noArrayIndexKey: bars are purely positional hourly buckets
              key={i}
              className="flex flex-1 items-end justify-center"
            >
              <span className="h-0.5 w-full max-w-2 rounded-full bg-gray-5" />
            </span>
          ))
        : bars.map((b, i) => {
            const total = b.success + b.error;
            const barHeight = Math.max(2, Math.round((total / max) * 32));
            const errorHeight = total > 0 ? Math.round((b.error / total) * barHeight) : 0;
            const successHeight = barHeight - errorHeight;
            return (
              <span
                // biome-ignore lint/suspicious/noArrayIndexKey: bars are purely positional hourly buckets
                key={i}
                className="flex flex-1 flex-col justify-end overflow-hidden rounded-[1px]"
                style={{ height: barHeight }}
              >
                {errorHeight > 0 && (
                  <span className="w-full bg-orange-9" style={{ height: errorHeight }} />
                )}
                {successHeight > 0 && (
                  <span className="w-full bg-accent-4" style={{ height: successHeight }} />
                )}
              </span>
            );
          })}
      <span className="pointer-events-none absolute inset-x-0 bottom-0 border-t border-dashed border-gray-5" />
    </div>
  );
}

function ConnectedResourcesPlaceholders({
  workspaceSlug,
  projectId,
}: {
  workspaceSlug: string;
  projectId: string;
}) {
  return (
    <div className="ml-4 border-l border-grayA-4 pl-5">
      <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
        <PlaceholderTile
          text="No keyspaces yet — create one to issue API keys."
          href={routes.projects.keyspaces({ workspaceSlug, projectId })}
          label="Go to keyspaces"
        />
        <PlaceholderTile
          text="No ratelimits yet — add one to protect an endpoint."
          href={routes.projects.ratelimits({ workspaceSlug, projectId })}
          label="Go to ratelimits"
        />
      </div>
    </div>
  );
}

function PlaceholderTile({ text, href, label }: { text: string; href: Route; label: string }) {
  return (
    <div className="flex flex-col items-start gap-2 rounded-lg border border-dashed border-grayA-4 p-4">
      <p className="text-[13px] text-gray-9">{text}</p>
      <Link href={href} className="text-xs font-medium text-accent-11 hover:text-accent-12">
        {label} →
      </Link>
    </div>
  );
}

function FootBand({
  usage,
  workspaceSlug,
}: {
  usage: OverviewProjectData["usage"];
  workspaceSlug: string;
}) {
  const requestFraction = usage.quota > 0 ? usage.billableTotal / usage.quota : null;
  const computeFraction =
    usage.hasComputePlan && usage.computeCredits > 0
      ? usage.computeSpend / usage.computeCredits
      : null;

  return (
    <div className="flex flex-col gap-4 rounded-lg border border-gray-4 bg-background p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="grid flex-1 grid-cols-1 gap-4 sm:grid-cols-2">
        <UsageMeter
          label="Requests"
          value={`${fmtCompact(usage.billableTotal)} / ${fmtCompact(usage.quota)}`}
          fraction={requestFraction}
          fillClassName="bg-accent-9"
        />
        {usage.hasComputePlan ? (
          <UsageMeter
            label="Compute"
            value={`$${usage.computeSpend.toFixed(2)} / $${usage.computeCredits.toFixed(2)}`}
            fraction={computeFraction}
            fillClassName="bg-info-9"
          />
        ) : (
          <div className="flex flex-col justify-center gap-1">
            <span className="text-[13px] text-gray-11">Compute</span>
            <Link
              href={routes.settings.billing({ workspaceSlug })}
              className="text-[13px] font-medium text-accent-11 hover:text-accent-12"
            >
              Add a plan →
            </Link>
          </div>
        )}
      </div>
      <span className="shrink-0 text-xs text-gray-9">{usage.daysLeft} days left</span>
    </div>
  );
}
