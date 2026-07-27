"use client";

import { DeploymentStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-badge";
import type { AppMock } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import type { HybridStyle } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/scenario";
import {
  GithubIcon,
  TerminalIcon,
  TrendArrow,
  fmtCompact,
  fmtInt,
  trendPct,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/ui";
import { routes } from "@/lib/navigation/routes";
import { cn } from "@/lib/utils";
import { CodeBranch, CodeCommit, Cube } from "@unkey/icons";
import { Badge, Button, Card } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import { CopyAgentPromptButton } from "./agent-prompt";
import { type DeploymentMock, fmtTimeAgo } from "./deployments-mock";
import { BleedRowChart, ContainedRowChart } from "./option-hybrid-chart";
import type { OverviewProjectData } from "./overview-data";
import { projectRequestSeries } from "./overview-mocks";

type KeyspaceStat = OverviewProjectData["keyspaces"][number];
type RatelimitStat = OverviewProjectData["ratelimits"][number];

// Andreas's FigJam idea, given its one proper execution: the metric stays the
// hero on the right, the trend moves into the background instead of competing
// with text for width. `hybridStyle` swaps between his full-bleed original and
// the safer contained rail treatment so both can be judged at page width.
export function OptionHybrid({
  data,
  hybridStyle,
}: { data: OverviewProjectData; hybridStyle: HybridStyle }) {
  const { workspaceSlug, scenario, project, keyspaces, ratelimits, usage, deployments } = data;

  const hasApps = project.apps.length > 0;
  const hasKeyspaces = keyspaces.length > 0;
  const hasRatelimits = ratelimits.length > 0;
  const isEmpty = !hasApps && !hasKeyspaces && !hasRatelimits;
  const showEmptyState = scenario === "new" || isEmpty;
  const showAppsSection = hasApps && scenario !== "migrated";
  const showDeployNudge = scenario === "migrated";

  const deployByApp = new Map(deployments.map((d) => [d.appId, d] as const));

  const totalRequests24h = keyspaces.reduce((sum, ks) => sum + ks.requests["24h"], 0);
  const avgValidPct = hasKeyspaces
    ? keyspaces.reduce((sum, ks) => sum + ks.validPct, 0) / keyspaces.length
    : null;
  const requestSeries = projectRequestSeries(
    project.id,
    48,
    Math.max(10, Math.round(totalRequests24h / 400)),
  );
  const requestDeltaPct = trendPct(requestSeries.map((p) => p.valid));

  return (
    <div className="flex flex-col gap-5">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <h1 className="truncate text-[22px] font-semibold tracking-tight leading-tight text-accent-12">
            {project.name}
          </h1>
          <div className="mt-1 truncate font-mono text-xs text-gray-9">{project.id}</div>
        </div>
        {!showEmptyState && (
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

      {!showEmptyState && (
        <div className="flex items-center gap-6">
          <SummaryStat
            label="Requests · 24h"
            value={fmtCompact(totalRequests24h)}
            deltaPct={requestDeltaPct}
          />
          <div className="border-l border-grayA-4 pl-6">
            <SummaryStat
              label="Success rate"
              value={avgValidPct === null ? "—" : `${avgValidPct.toFixed(1)}%`}
            />
          </div>
          <div className="border-l border-grayA-4 pl-6">
            {usage.hasComputePlan ? (
              <SummaryStat
                label="Compute"
                value={`$${usage.computeSpend.toFixed(0)} / $${usage.computeCredits.toFixed(0)}`}
              />
            ) : (
              <Link href={routes.settings.billing({ workspaceSlug })} className="group block">
                <div className="text-xs text-gray-9">Compute</div>
                <div className="mt-0.5 text-lg font-semibold tabular-nums text-accent-11 group-hover:text-accent-12">
                  No plan <span className="text-xs font-normal">· Billing →</span>
                </div>
              </Link>
            )}
          </div>
        </div>
      )}

      {showEmptyState ? (
        <NewProjectState workspaceSlug={workspaceSlug} projectId={project.id} />
      ) : (
        <>
          {showDeployNudge && (
            <DeployNudgeRow workspaceSlug={workspaceSlug} projectId={project.id} />
          )}

          {showAppsSection && (
            <ListSection title="Apps">
              {project.apps.map((app) => (
                <AppRow
                  key={app.id}
                  app={app}
                  dep={deployByApp.get(app.id)}
                  href={routes.projects.apps.overview({
                    workspaceSlug,
                    projectId: project.id,
                    appId: app.id,
                  })}
                />
              ))}
            </ListSection>
          )}

          {hasKeyspaces && (
            <ListSection title="Keyspaces">
              {keyspaces.map((ks) => (
                <KeyspaceRow
                  key={ks.id}
                  keyspace={ks}
                  hybridStyle={hybridStyle}
                  href={routes.apis.detail({ workspaceSlug, apiId: ks.id })}
                />
              ))}
            </ListSection>
          )}

          {hasRatelimits && (
            <ListSection title="Ratelimits">
              {ratelimits.map((rl) => (
                <RatelimitRow
                  key={rl.id}
                  ratelimit={rl}
                  hybridStyle={hybridStyle}
                  href={routes.ratelimits.detail({ workspaceSlug, namespaceId: rl.id })}
                />
              ))}
            </ListSection>
          )}
        </>
      )}
    </div>
  );
}

function SummaryStat({
  label,
  value,
  deltaPct,
}: { label: string; value: string; deltaPct?: number }) {
  return (
    <div>
      <div className="text-xs text-gray-9">{label}</div>
      <div className="mt-0.5 flex items-center gap-1.5">
        <span className="text-lg font-semibold tabular-nums text-accent-12">{value}</span>
        {deltaPct !== undefined && deltaPct !== 0 && <DeltaBadge deltaPct={deltaPct} />}
      </div>
    </div>
  );
}

function DeltaBadge({ deltaPct }: { deltaPct: number }) {
  return (
    <span
      className={cn(
        "flex items-center gap-0.5 text-[11px] tabular-nums",
        deltaPct > 0 ? "text-success-11" : "text-gray-11",
      )}
    >
      <TrendArrow up={deltaPct > 0} className="size-2.5" />
      {Math.abs(deltaPct)}%
    </span>
  );
}

function ListSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-lg border border-grayA-4 overflow-hidden">
      <div className="px-4 py-3 border-b border-grayA-4">
        <span className="text-sm font-medium text-accent-12">{title}</span>
      </div>
      <div className="divide-y divide-grayA-4">{children}</div>
    </div>
  );
}

// Deploy nudge for the migrated scenario: no apps exist yet, so this stands in
// for the missing Apps section as its own single-row container rather than a
// card, keeping the same list-row visual language as everything below it.
function DeployNudgeRow({
  workspaceSlug,
  projectId,
}: { workspaceSlug: string; projectId: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-grayA-4 px-4 py-3.5">
      <div className="flex min-w-0 items-center gap-3">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-gray-3 text-gray-11">
          <GithubIcon className="size-3.5" />
        </span>
        <span className="truncate text-[13px] font-medium text-accent-12">
          Deploy your app — connect a repo or hand off to your agent
        </span>
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

function AppRow({
  app,
  dep,
  href,
}: { app: AppMock; dep: DeploymentMock | undefined; href: Route }) {
  return (
    <Link
      href={href}
      className="flex items-center justify-between gap-3 px-4 py-3.5 hover:bg-grayA-2"
    >
      <div className="flex min-w-0 items-center gap-3">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-gray-3 text-gray-11">
          {app.source === "github" ? (
            <GithubIcon className="size-3.5" />
          ) : (
            <TerminalIcon className="size-3.5" />
          )}
        </span>
        <div className="min-w-0">
          <div className="truncate text-[13px] font-medium text-accent-12">{app.name}</div>
          {dep ? (
            <div className="mt-0.5 flex items-center gap-1.5 text-xs text-gray-9">
              <CodeBranch iconSize="sm-regular" className="shrink-0" />
              <span className="truncate font-mono text-accent-12">{dep.branch}</span>
              <CodeCommit iconSize="sm-regular" className="shrink-0" />
              <span className="font-mono text-accent-12">{dep.sha.slice(0, 7)}</span>
            </div>
          ) : (
            <div className="mt-0.5 text-xs text-gray-9">Not deployed</div>
          )}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {dep ? (
          <>
            <DeploymentStatusBadge status={dep.status} />
            <span className="w-14 text-right text-xs text-gray-9 tabular-nums">
              {fmtTimeAgo(dep.timeAgoMin)}
            </span>
          </>
        ) : (
          <Badge variant="secondary" size="sm">
            No deployments
          </Badge>
        )}
      </div>
    </Link>
  );
}

// The concept row: metric on the right stays the hero, the trend lives in the
// background (bleed) or a small contained box (contained) instead of a
// foreground chart competing with the text for space.
function MetricRow({
  title,
  subtitle,
  value,
  label,
  deltaPct,
  valid,
  error,
  hybridStyle,
  href,
}: {
  title: string;
  subtitle: string;
  value: string;
  label: string;
  deltaPct: number;
  valid: number[];
  error: number[];
  hybridStyle: HybridStyle;
  href: Route;
}) {
  const isBleed = hybridStyle === "bleed";
  return (
    <Link
      href={href}
      className="group relative flex min-h-[72px] items-center gap-4 overflow-hidden px-5 py-4 hover:bg-grayA-2"
    >
      {isBleed && (
        <>
          <BleedRowChart
            valid={valid}
            error={error}
            className="pointer-events-none absolute inset-x-0 bottom-0 h-[46px] w-full"
          />
          <div className="pointer-events-none absolute inset-x-0 top-0 h-8 bg-linear-to-b from-background/85 to-transparent" />
        </>
      )}

      <div className="relative z-10 min-w-0 flex-1">
        <div className="truncate text-[13px] font-medium text-accent-12">{title}</div>
        <div className="mt-0.5 truncate text-xs text-gray-9">{subtitle}</div>
      </div>

      {!isBleed && (
        <ContainedRowChart
          valid={valid}
          error={error}
          className="relative z-10 h-9 w-44 shrink-0"
        />
      )}

      <div className="relative z-10 flex shrink-0 flex-col items-end gap-0.5">
        <div className="flex items-center gap-1.5">
          <span className="text-xl font-semibold tabular-nums leading-tight text-accent-12">
            {value}
          </span>
          {deltaPct !== 0 && <DeltaBadge deltaPct={deltaPct} />}
        </div>
        <span className="text-xs text-gray-9">{label}</span>
      </div>
    </Link>
  );
}

// The generator's own error rate is independent random noise, not the row's
// actual validPct/blockedPct — re-split its volume shape by the real stat
// (same idea as option-stats.tsx's splitBuckets) so the rendered error line
// actually reflects the percentage shown in the row's subtitle.
function splitByRealRate(
  series: { valid: number; error: number }[],
  successPct: number,
): { valid: number; error: number }[] {
  const pct = Math.min(100, Math.max(0, successPct)) / 100;
  return series.map((p) => {
    const volume = p.valid + p.error;
    const valid = Math.round(volume * pct);
    return { valid, error: Math.max(0, volume - valid) };
  });
}

function KeyspaceRow({
  keyspace,
  hybridStyle,
  href,
}: { keyspace: KeyspaceStat; hybridStyle: HybridStyle; href: Route }) {
  const base = Math.max(8, Math.round(Math.sqrt(keyspace.requests["24h"])));
  const series = splitByRealRate(projectRequestSeries(keyspace.id, 48, base), keyspace.validPct);
  const valid = series.map((p) => p.valid);
  const error = series.map((p) => p.error);
  return (
    <MetricRow
      title={keyspace.name}
      subtitle={`${fmtInt(keyspace.keyCount)} keys · ${keyspace.validPct}% valid`}
      value={fmtCompact(keyspace.requests["24h"])}
      label="requests · 24h"
      deltaPct={trendPct(valid)}
      valid={valid}
      error={error}
      hybridStyle={hybridStyle}
      href={href}
    />
  );
}

function RatelimitRow({
  ratelimit,
  hybridStyle,
  href,
}: { ratelimit: RatelimitStat; hybridStyle: HybridStyle; href: Route }) {
  const base = Math.max(8, Math.round(Math.sqrt(ratelimit.checks["24h"])));
  const series = splitByRealRate(
    projectRequestSeries(ratelimit.id, 48, base),
    100 - ratelimit.blockedPct,
  );
  const valid = series.map((p) => p.valid);
  const error = series.map((p) => p.error);
  return (
    <MetricRow
      title={ratelimit.name}
      subtitle={`${ratelimit.blockedPct}% blocked`}
      value={fmtCompact(ratelimit.checks["24h"])}
      label="checks · 24h"
      deltaPct={trendPct(valid)}
      valid={valid}
      error={error}
      hybridStyle={hybridStyle}
      href={href}
    />
  );
}

function NewProjectState({
  workspaceSlug,
  projectId,
}: { workspaceSlug: string; projectId: string }) {
  return (
    <>
      <Card className="flex flex-col items-center gap-3 py-14 text-center">
        <span className="flex size-11 items-center justify-center rounded-lg bg-gray-3 text-gray-11">
          <Cube iconSize="lg-regular" />
        </span>
        <div>
          <h3 className="text-[15px] font-semibold text-accent-12">Nothing here yet</h3>
          <p className="mt-1 max-w-xs text-[13px] text-gray-9">
            Deploy an app or bring in a keyspace to see it show up here.
          </p>
        </div>
        <div className="mt-1 flex items-center gap-2">
          <Button
            size="md"
            variant="primary"
            render={<Link href={routes.projects.apps.new({ workspaceSlug, projectId })} />}
          >
            Connect GitHub
          </Button>
          <CopyAgentPromptButton />
        </div>
      </Card>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <GhostRow label="Apps will appear here" />
        <GhostRow label="Keyspaces will appear here" />
        <GhostRow label="Ratelimits will appear here" />
      </div>
    </>
  );
}

function GhostRow({ label }: { label: string }) {
  return (
    <div className="rounded-lg border border-dashed border-grayA-4 px-4 py-6 text-center text-xs text-gray-9">
      {label}
    </div>
  );
}
