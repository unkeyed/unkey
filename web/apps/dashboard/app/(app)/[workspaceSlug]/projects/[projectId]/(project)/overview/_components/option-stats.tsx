"use client";

import { PROMPT } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/agent-setup";
import {
  GaugeIcon,
  GithubIcon,
  KeyIcon,
  TerminalIcon,
  fmtCompact,
  fmtInt,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/ui";
import { StatsListCard, type StatsListCardBucket } from "@/components/stats-list-card";
import { routes } from "@/lib/navigation/routes";
import { cn } from "@/lib/utils";
import { CodeCommit, Cube } from "@unkey/icons";
import { Button, Card, CopyButton, Separator } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { type ComponentType, type ReactNode, useState } from "react";
import { fmtTimeAgo } from "./deployments-mock";
import type { OverviewProjectData } from "./overview-data";
import { type ActivityEvent, activityForProject, projectRequestSeries } from "./overview-mocks";

const HOURS_IN_DAY = 24;

// Supabase project-home inspired: big name + url-style id up top, a labeled fact
// strip, then the "product tiles" (real StatsListCard, the same component the
// api/ratelimit list pages use) as the centerpiece, so the page reads as live
// telemetry instead of a wireframe of boxes.
export function OptionStats({ data }: { data: OverviewProjectData }) {
  const { workspaceSlug, project, keyspaces, ratelimits, usage, deployments } = data;
  const isNew = data.scenario === "new";
  const hasApps = project.apps.length > 0;

  const githubApp = project.apps.find((app) => app.source === "github");
  const latestDeploy = deployments.length
    ? deployments.reduce((latest, d) => (d.timeAgoMin < latest.timeAgoMin ? d : latest))
    : undefined;
  const anyFailed = deployments.some((d) => d.status === "failed");
  const anyBuilding = deployments.some((d) => d.status === "building");
  const status: { label: string; dot: string } = hasApps
    ? anyFailed
      ? { label: "Attention", dot: "bg-error-9" }
      : anyBuilding
        ? { label: "Building", dot: "bg-warning-9" }
        : { label: "Healthy", dot: "bg-success-9" }
    : { label: "—", dot: "bg-gray-9" };

  const totalKeyspaceReq = keyspaces.reduce((sum, ks) => sum + ks.requests["24h"], 0);
  const totalChecks = ratelimits.reduce((sum, rl) => sum + rl.checks["24h"], 0);
  const totalKeys = keyspaces.reduce((sum, ks) => sum + ks.keyCount, 0);
  const avgValidPct = keyspaces.length
    ? keyspaces.reduce((sum, ks) => sum + ks.validPct, 0) / keyspaces.length
    : 100;
  const avgBlockedPct = ratelimits.length
    ? ratelimits.reduce((sum, rl) => sum + rl.blockedPct, 0) / ratelimits.length
    : 0;
  const requestsTotal = totalKeyspaceReq + totalChecks;
  const successRatePct = keyspaces.length
    ? avgValidPct
    : ratelimits.length
      ? 100 - avgBlockedPct
      : 100;

  const requestBuckets = toBuckets(
    projectRequestSeries(
      `requests-${project.id}`,
      HOURS_IN_DAY,
      Math.max(1, requestsTotal / HOURS_IN_DAY),
    ),
  );
  const verificationBuckets = splitBuckets(
    projectRequestSeries(
      `verifications-${project.id}`,
      HOURS_IN_DAY,
      Math.max(1, totalKeyspaceReq / HOURS_IN_DAY),
    ),
    avgValidPct,
  );
  const ratelimitBuckets = splitBuckets(
    projectRequestSeries(
      `ratelimit-checks-${project.id}`,
      HOURS_IN_DAY,
      Math.max(1, totalChecks / HOURS_IN_DAY),
    ),
    100 - avgBlockedPct,
  );

  const activity = activityForProject(project, keyspaces, ratelimits).slice(0, 4);

  const repoTile: ConnectTileDef = {
    key: "repo",
    label: hasApps ? "Deploy an app" : "Connect a repo",
    sub: hasApps ? "Add another app" : "Bring code into this project",
    icon: GithubIcon,
    href: routes.projects.apps.new({ workspaceSlug, projectId: project.id }),
  };
  const keyspacesTile: ConnectTileDef = {
    key: "keyspaces",
    label: "Keyspaces",
    sub: keyspaces.length ? `${fmtInt(keyspaces.length)} active` : "Manage API keys",
    icon: KeyIcon,
    href: routes.projects.keyspaces({ workspaceSlug, projectId: project.id }),
  };
  const ratelimitsTile: ConnectTileDef = {
    key: "ratelimits",
    label: "Ratelimits",
    sub: ratelimits.length ? `${fmtInt(ratelimits.length)} active` : "Protect an endpoint",
    icon: GaugeIcon,
    href: routes.projects.ratelimits({ workspaceSlug, projectId: project.id }),
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <h2 className="truncate text-[22px] font-semibold leading-tight tracking-tight text-accent-12">
            {project.name}
          </h2>
          <div className="mt-1 flex items-center gap-1 text-xs text-gray-9">
            <span className="truncate font-mono">{project.id}</span>
            <CopyButton
              value={project.id}
              variant="ghost"
              className="size-5 shrink-0 [&_svg]:size-3"
            />
          </div>
        </div>
        {!isNew && (
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

      {/* A row of em-dashes says nothing — the fact strip only earns its place
          once the project has apps to report on. */}
      {hasApps && (
        <div className="flex flex-wrap items-start gap-x-6 gap-y-4 rounded-lg border border-grayA-4 p-5">
          <FactCell label="Status" first>
            <span className={cn("size-2 shrink-0 rounded-full", status.dot)} />
            {status.label}
          </FactCell>
          <FactCell label="Compute">
            {usage.hasComputePlan
              ? `$${usage.computeSpend.toFixed(2)} of $${usage.computeCredits.toFixed(0)} credits`
              : "No plan"}
          </FactCell>
          <FactCell label="Github">
            {githubApp ? (
              <>
                <GithubIcon className="size-3.5 shrink-0 text-gray-9" />
                <span className="truncate font-mono">unkey/{githubApp.name}</span>
              </>
            ) : (
              "Not connected"
            )}
          </FactCell>
          <FactCell label="Last deploy">
            {latestDeploy ? (
              <>
                <CodeCommit className="size-3.5 shrink-0 text-gray-9" />
                <span className="font-mono">{latestDeploy.sha.slice(0, 7)}</span>
                <span className="text-gray-9">· {fmtTimeAgo(latestDeploy.timeAgoMin)}</span>
              </>
            ) : (
              "—"
            )}
          </FactCell>
          <FactCell label="Apps">
            <Cube className="size-3.5 shrink-0 text-gray-9" />
            {fmtInt(project.appCount)}
          </FactCell>
        </div>
      )}

      {isNew ? (
        // Same grid, same card bones as the live tiles — each slot pitches its
        // own setup instead of pretending to chart nothing.
        <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
          <SetupCard
            title="Requests"
            description="Traffic across your apps will chart here."
            actionLabel="Connect a repo"
            href={routes.projects.apps.new({ workspaceSlug, projectId: project.id })}
          />
          <SetupCard
            title="Verifications"
            description="Key usage across your keyspaces will chart here."
            actionLabel="Create a keyspace"
            href={routes.projects.keyspaces({ workspaceSlug, projectId: project.id })}
          />
          <SetupCard
            title="Ratelimits"
            description="Checks and blocks will chart here."
            actionLabel="Add a ratelimit"
            href={routes.projects.ratelimits({ workspaceSlug, projectId: project.id })}
          />
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3">
          <StatsListCard
            href={routes.projects.apps.list({ workspaceSlug, projectId: project.id })}
            ariaLabel="View app requests"
            title="Requests"
            subtitle={hasApps || keyspaces.length === 0 ? "All apps · 24h" : "All keyspaces · 24h"}
            buckets={requestBuckets}
            isLoading={false}
            isError={false}
            labels={{ success: "ok", error: "errors" }}
            footerLeft={
              <span className="tabular-nums">{fmtCompact(requestsTotal)} total · 24h</span>
            }
          />
          <StatsListCard
            href={routes.projects.keyspaces({ workspaceSlug, projectId: project.id })}
            ariaLabel="View key verifications"
            title="Verifications"
            subtitle={
              keyspaces.length
                ? `${keyspaces.length} keyspace${keyspaces.length === 1 ? "" : "s"}`
                : undefined
            }
            buckets={verificationBuckets}
            isLoading={false}
            isError={false}
            labels={{ success: "Valid", error: "Invalid" }}
            footerLeft={<span className="tabular-nums">{fmtInt(totalKeys)} keys</span>}
          />
          <StatsListCard
            href={routes.projects.ratelimits({ workspaceSlug, projectId: project.id })}
            ariaLabel="View ratelimit checks"
            title="Ratelimits"
            subtitle={
              ratelimits.length
                ? `${ratelimits.length} namespace${ratelimits.length === 1 ? "" : "s"}`
                : undefined
            }
            buckets={ratelimitBuckets}
            isLoading={false}
            isError={false}
            labels={{ success: "Passed", error: "Blocked" }}
            footerLeft={<span className="tabular-nums">{fmtCompact(totalChecks)} checks</span>}
          />
        </div>
      )}

      <div>
        <span className="text-sm font-medium text-accent-12">Get connected</span>
        <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
          {hasApps ? (
            <>
              <ConnectTileLink tile={repoTile} />
              <ConnectTileLink tile={keyspacesTile} />
              <ConnectTileLink tile={ratelimitsTile} />
              <AgentTile />
            </>
          ) : (
            <>
              <ConnectTileLink tile={repoTile} />
              <AgentTile />
              <ConnectTileLink tile={keyspacesTile} />
              <ConnectTileLink tile={ratelimitsTile} />
            </>
          )}
        </div>
      </div>

      {isNew ? (
        <Card className="flex flex-col overflow-hidden">
          <div className="border-b border-grayA-4 px-4 py-3">
            <span className="text-sm font-medium text-accent-12">Getting started</span>
          </div>
          <div className="flex flex-col md:flex-row md:divide-x md:divide-grayA-4">
            {GETTING_STARTED_STEPS.map((step, i) => (
              <div key={step} className="flex flex-1 items-center gap-2.5 px-4 py-3">
                <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-gray-3 text-[11px] font-medium text-gray-9">
                  {i + 1}
                </span>
                <span className="whitespace-nowrap text-[13px] text-gray-11">{step}</span>
              </div>
            ))}
          </div>
        </Card>
      ) : (
        <Card className="flex flex-col gap-6 p-5 sm:flex-row sm:items-center">
          <div className="flex items-center gap-6">
            <div className="flex flex-col gap-1">
              <span className="text-2xl font-semibold tabular-nums text-accent-12">
                {fmtCompact(requestsTotal)}
              </span>
              <span className="text-xs text-gray-9">requests · 24h</span>
            </div>
            <Separator orientation="vertical" className="h-10" />
            <div className="flex flex-col gap-1">
              <span className="text-2xl font-semibold tabular-nums text-accent-12">
                {successRatePct.toFixed(1)}%
              </span>
              <span className="text-xs text-gray-9">success rate</span>
            </div>
            {latestDeploy ? (
              <>
                <Separator orientation="vertical" className="h-10" />
                <div className="flex flex-col gap-1">
                  <span className="flex items-center gap-1.5 text-[13px] font-medium text-accent-12">
                    <CodeCommit className="size-3.5 shrink-0 text-gray-9" />
                    <span className="font-mono">{latestDeploy.sha.slice(0, 7)}</span>
                  </span>
                  <span className="text-xs text-gray-9">
                    {latestDeploy.appName} · {fmtTimeAgo(latestDeploy.timeAgoMin)}
                  </span>
                </div>
              </>
            ) : null}
          </div>
          <Separator orientation="vertical" className="hidden h-10 sm:block" />
          <div className="min-w-0 flex-1">
            <div className="mb-2 text-xs font-medium text-gray-9">Recent activity</div>
            <ul className="flex flex-col gap-1.5">
              {activity.map((event) => (
                <ActivityRow key={event.id} event={event} />
              ))}
            </ul>
          </div>
        </Card>
      )}
    </div>
  );
}

type ConnectTileDef = {
  key: string;
  label: string;
  sub: string;
  icon: ComponentType<{ className?: string }>;
  href: Route;
};

function ConnectTileLink({ tile }: { tile: ConnectTileDef }) {
  return (
    <Link
      href={tile.href}
      className="flex flex-col gap-2 rounded-lg border border-grayA-4 p-3 hover:border-grayA-7"
    >
      <tile.icon className="size-4 text-gray-9" />
      <div className="min-w-0">
        <div className="text-[13px] font-medium text-accent-12">{tile.label}</div>
        <div className="mt-0.5 truncate text-xs text-gray-9">{tile.sub}</div>
      </div>
    </Link>
  );
}

function AgentTile() {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(PROMPT);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1800);
  };
  return (
    <button
      type="button"
      onClick={copy}
      className="flex flex-col gap-2 rounded-lg border border-grayA-4 p-3 text-left hover:border-grayA-7"
    >
      <TerminalIcon className="size-4 text-gray-9" />
      <div className="min-w-0">
        <div className="text-[13px] font-medium text-accent-12">Set up your agent</div>
        <div className="mt-0.5 truncate text-xs text-gray-9">
          {copied ? "Copied — paste into your agent" : "Claude, Cursor, Codex"}
        </div>
      </div>
    </button>
  );
}

const GETTING_STARTED_STEPS = [
  "Deploy your app",
  "Create a keyspace",
  "Add a ratelimit",
  "Invite your team",
];

// Gentle bell curve for the ghost chart — obviously decorative (uniform, faint),
// so it reads "a chart lives here" without impersonating real zero data.
const GHOST_HEIGHTS = [
  10, 14, 18, 24, 30, 36, 40, 44, 46, 44, 40, 34, 38, 42, 40, 34, 28, 22, 26, 30, 26, 20, 14, 10,
];

// Mirrors the StatsListCard shell (padding, title scale, h-12 chart well,
// footer) so the new-project grid keeps the active page's exact bones.
function SetupCard({
  title,
  description,
  actionLabel,
  href,
}: {
  title: string;
  description: string;
  actionLabel: string;
  href: Route;
}) {
  return (
    <Link
      href={href}
      className="group relative flex h-full w-full flex-col gap-5 rounded-lg border border-grayA-4 p-5 transition-all duration-300 hover:border-grayA-7"
    >
      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium leading-[14px] text-accent-12">{title}</span>
        <span className="truncate text-xs leading-[12px] text-gray-11">{description}</span>
      </div>
      <div className="relative h-12 w-full">
        <div className="flex h-full w-full items-end gap-[3px]">
          {GHOST_HEIGHTS.map((h, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: positional bars
            <div
              key={i}
              className="flex-1 rounded-t-[1px] bg-grayA-3"
              style={{ height: `${h}%` }}
            />
          ))}
        </div>
        <div className="pointer-events-none absolute inset-x-0 bottom-0 border-t border-dashed border-gray-5" />
      </div>
      <div className="flex items-center gap-1 text-xs font-medium text-accent-11 group-hover:text-accent-12">
        {actionLabel}
        <span aria-hidden>→</span>
      </div>
    </Link>
  );
}

function FactCell({
  label,
  first,
  children,
}: {
  label: string;
  first?: boolean;
  children: ReactNode;
}) {
  return (
    <div className={cn("flex flex-col", !first && "border-l border-grayA-4 pl-6")}>
      <span className="text-[10px] font-medium uppercase tracking-wide text-gray-9">{label}</span>
      <span className="mt-1 flex items-center gap-1.5 text-[13px] text-accent-12">{children}</span>
    </div>
  );
}

const ACTIVITY_DOT: Record<ActivityEvent["kind"], string> = {
  deploy: "bg-accent-9",
  key: "bg-success-9",
  ratelimit: "bg-warning-9",
  member: "bg-gray-9",
  domain: "bg-gray-9",
};

function ActivityRow({ event }: { event: ActivityEvent }) {
  return (
    <li className="flex items-center gap-2 text-xs text-gray-11">
      <span className={cn("size-1.5 shrink-0 rounded-full", ACTIVITY_DOT[event.kind])} />
      <span className="truncate">{event.text}</span>
      <span className="ml-auto shrink-0 tabular-nums text-gray-9">
        {fmtTimeAgo(event.timeAgoMin)}
      </span>
    </li>
  );
}

function hourLabel(index: number, points: number): string {
  const hoursAgo = points - 1 - index;
  const d = new Date();
  d.setHours(d.getHours() - hoursAgo, 0, 0, 0);
  return `${String(d.getHours()).padStart(2, "0")}:00`;
}

function toBuckets(series: { valid: number; error: number }[]): StatsListCardBucket[] {
  return series.map((point, i) => ({
    displayX: hourLabel(i, series.length),
    success: point.valid,
    error: point.error,
  }));
}

// Reuses the generator's realistic hourly shape as raw traffic volume, then
// re-splits each bucket by a real stat (validPct / 100-blockedPct) instead of
// the generator's own randomized error rate — so the chart's error share
// actually reflects the project's data.
function splitBuckets(
  series: { valid: number; error: number }[],
  successPct: number,
): StatsListCardBucket[] {
  const pct = Math.min(100, Math.max(0, successPct)) / 100;
  return series.map((point, i) => {
    const volume = point.valid + point.error;
    const success = Math.round(volume * pct);
    return {
      displayX: hourLabel(i, series.length),
      success,
      error: Math.max(0, volume - success),
    };
  });
}
