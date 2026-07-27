"use client";

import { AppActions } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/apps-list/app-actions";
import { DeploymentStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-badge";
import { PROMPT } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/agent-setup";
import type { Mark } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/marks";
import {
  RailListShell,
  RailRow,
  type RowItem,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/rail";
import type {
  OverviewLayout,
  RowVariant,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/scenario";
import { fmtCompact, fmtInt } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/ui";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { cn } from "@/lib/utils";
import {
  BookBookmark,
  Chats,
  Check,
  ChevronDown,
  CodeBranch,
  Cube,
  Discord,
  Dots,
  Github,
  Layers2,
  Plus,
  Terminal,
  XMark,
} from "@unkey/icons";
import { Button, Card } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { useParams } from "next/navigation";
import { type ReactNode, useState } from "react";
import { type DeploymentMock, fmtTimeAgo } from "./deployments-mock";
import type { OverviewProjectData } from "./overview-data";
import { projectRequestSeries } from "./overview-mocks";

const DISCORD_URL = "https://unkey.com/discord";
const SUPPORT_URL = "https://unkey.com/support";
const TEMPLATES_URL = "https://unkey.com/templates";
const DOCS_URL = "https://unkey.com/docs";

// One template for the header and the rows — the whole point of the table
// treatment is that columns line up, so it can only be declared once.
const APP_GRID = "grid grid-cols-[minmax(0,1fr)_120px_104px_76px_32px] items-center gap-3";

// Matches the projects rail: divided rows with a bar mark, and the same
// valid/success bar color the api and ratelimit list pages use.
const ROW_VARIANT: RowVariant = "list";
const ROW_MARK: Mark = "bars";
const CHART_STROKE = "hsl(var(--accent-4))";
const HOURS_IN_DAY = 24;

// One generated series per resource, seeded on its id: the chart, the value and
// the trend all come from the same 24 hourly buckets, so they can't contradict
// each other the way a hand-written spark array and a separate total did.
function hourlySeries(seed: string, total24h: number, errorPct: number) {
  const pct = Math.min(100, Math.max(0, errorPct)) / 100;
  const raw = projectRequestSeries(seed, HOURS_IN_DAY, Math.max(1, total24h / HOURS_IN_DAY));
  const volume = raw.reduce((sum, b) => sum + b.valid + b.error, 0);
  const generatedErrors = raw.reduce((sum, b) => sum + b.error, 0) || 1;
  // Scale the generator's error series so its total lands on the resource's real
  // invalid/blocked rate. Re-deriving each bucket from the flat rate instead
  // paints an identical orange cap on every bar, which is the giveaway that no
  // real error data was involved.
  const scale = (volume * pct) / generatedErrors;
  const buckets = raw.map(({ valid, error }) => {
    const total = valid + error;
    const errored = Math.min(total, Math.round(error * scale));
    return { valid: Math.max(0, total - errored), error: errored };
  });
  return {
    buckets,
    totals: buckets.map((b) => b.valid + b.error),
    total: buckets.reduce((sum, b) => sum + b.valid + b.error, 0),
  };
}

function useProjectScope(): { workspaceSlug: string; projectId: string } {
  const workspace = useWorkspaceNavigation();
  const params = useParams<{ projectId: string }>();
  return { workspaceSlug: workspace.slug, projectId: params.projectId };
}

// Apps are the page. No resource-count strip on top: the apps section already
// says how many apps there are, and the rail lists the actual keyspaces and
// ratelimits rather than counting them.
export function ProjectOverview({
  data,
  layout = "collapse",
}: {
  data: OverviewProjectData;
  layout?: OverviewLayout;
}) {
  const { workspaceSlug, project, keyspaces, ratelimits, deployments } = data;
  const [dismissed, setDismissed] = useState(false);
  const scope = { workspaceSlug, projectId: project.id };
  const hasApps = project.apps.length > 0;

  const apps = (
    <AppsCard
      appsHref={routes.projects.apps.list(scope)}
      newAppHref={routes.projects.apps.new(scope)}
      deployments={deployments}
      hasApps={hasApps}
      keyspaceCount={keyspaces.length}
      ratelimitCount={ratelimits.length}
    />
  );

  // A resource card only renders once it has rows, so a brand-new project starts
  // without a rail full of zeroes.
  const resources = (
    <ResourceLists
      workspaceSlug={workspaceSlug}
      projectId={project.id}
      keyspaces={keyspaces}
      ratelimits={ratelimits}
      wide={layout === "column"}
    />
  );

  if (layout === "column") {
    return (
      <div className="flex flex-col gap-6">
        {!dismissed && (
          <GettingStarted data={data} onDismiss={() => setDismissed(true)} columns={3} />
        )}
        {apps}
        <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">{resources}</div>
        <HelpCard wide />
      </div>
    );
  }

  return (
    <div className="flex items-start gap-6 max-lg:flex-col">
      <div className="flex min-w-0 flex-1 flex-col gap-6 max-lg:order-2 max-lg:w-full">
        {!dismissed && <GettingStarted data={data} onDismiss={() => setDismissed(true)} />}
        {apps}
      </div>
      <aside className="flex w-full shrink-0 flex-col gap-4 max-lg:order-1 lg:w-[260px]">
        {resources}
        <HelpCard />
      </aside>
    </div>
  );
}

/* ----------------------------------------------------------- resource lists */

// One card per resource type with a row per keyspace / namespace, using the
// same shell and row the projects rail uses. No project name in the subtitle:
// on a project page it would repeat on every row.
function ResourceLists({
  workspaceSlug,
  projectId,
  keyspaces,
  ratelimits,
  wide = false,
}: {
  workspaceSlug: string;
  projectId: string;
  keyspaces: OverviewProjectData["keyspaces"];
  ratelimits: OverviewProjectData["ratelimits"];
  /** Full-width cards have room for the 24h volume; rail width does not. */
  wide?: boolean;
}) {
  const rowVariant: RowVariant = wide ? "metric" : ROW_VARIANT;

  const keyspaceRows: RowItem[] = keyspaces.map((ks) => {
    const series = hourlySeries(`keyspace-${ks.id}`, ks.requests["24h"], 100 - ks.validPct);
    return {
      id: ks.id,
      title: ks.name,
      subtitle: `${fmtInt(ks.keyCount)} keys`,
      value: fmtCompact(series.total),
      spark: series.totals,
      buckets: series.buckets,
      errorRatio: (100 - ks.validPct) / 100,
      stroke: CHART_STROKE,
      kind: "keyspace",
      href: routes.apis.detail({ workspaceSlug, apiId: ks.id }),
    };
  });
  const ratelimitRows: RowItem[] = ratelimits.map((rl) => {
    const series = hourlySeries(`ratelimit-${rl.id}`, rl.checks["24h"], rl.blockedPct);
    return {
      id: rl.id,
      title: rl.name,
      subtitle: `${rl.blockedPct}% blocked`,
      value: fmtCompact(series.total),
      spark: series.totals,
      buckets: series.buckets,
      errorRatio: rl.blockedPct / 100,
      stroke: CHART_STROKE,
      kind: "ratelimit",
      href: routes.ratelimits.detail({ workspaceSlug, namespaceId: rl.id }),
    };
  });

  return (
    <>
      {keyspaceRows.length > 0 && (
        <RailListShell
          title="Keyspaces"
          variant={ROW_VARIANT}
          subtitle="24h"
          viewAllHref={routes.projects.keyspaces({ workspaceSlug, projectId })}
        >
          {keyspaceRows.map((item) => (
            <RailRow key={item.id} item={item} variant={rowVariant} mark={ROW_MARK} />
          ))}
        </RailListShell>
      )}
      {ratelimitRows.length > 0 && (
        <RailListShell
          title="Ratelimits"
          variant={ROW_VARIANT}
          subtitle="24h"
          viewAllHref={routes.projects.ratelimits({ workspaceSlug, projectId })}
        >
          {ratelimitRows.map((item) => (
            <RailRow key={item.id} item={item} variant={rowVariant} mark={ROW_MARK} />
          ))}
        </RailListShell>
      )}
    </>
  );
}

/* ---------------------------------------------------------- getting started */

type Step = {
  key: string;
  label: string;
  hint: string;
  /** Shown instead of the hint once complete — the fact, not the pitch. */
  done: (data: OverviewProjectData) => string;
  href?: (scope: { workspaceSlug: string; projectId: string }) => Route;
  /** Copies to the clipboard instead of navigating. */
  copy?: string;
};

const STEPS: Step[] = [
  {
    key: "agent",
    label: "Set up with your agent",
    hint: "Copy a prompt · Claude, Cursor, Codex",
    done: () => "Prompt copied",
    copy: PROMPT,
  },
  {
    key: "app",
    label: "Deploy an app",
    hint: "Bring code into this project",
    done: ({ project }) =>
      `${fmtInt(project.apps.length)} app${project.apps.length === 1 ? "" : "s"} deployed`,
    href: (scope) => routes.projects.apps.new(scope),
  },
  {
    key: "keyspace",
    label: "Create a keyspace",
    hint: "Issue and verify API keys",
    done: ({ keyspaces }) =>
      `${fmtInt(keyspaces.length)} keyspace${keyspaces.length === 1 ? "" : "s"} active`,
    href: (scope) => routes.projects.keyspaces(scope),
  },
  {
    key: "ratelimit",
    label: "Add a ratelimit",
    hint: "Protect your API from abuse",
    done: ({ ratelimits }) =>
      `${fmtInt(ratelimits.length)} namespace${ratelimits.length === 1 ? "" : "s"} active`,
    href: (scope) => routes.projects.ratelimits(scope),
  },
  {
    key: "domain",
    label: "Add a custom domain",
    hint: "Serve on your own hostname",
    done: () => "Configured",
    href: (scope) => routes.projects.settings(scope),
  },
  {
    key: "team",
    label: "Invite your team",
    hint: "Share this workspace",
    done: () => "Invites sent",
    href: ({ workspaceSlug }) => routes.settings.team({ workspaceSlug }),
  },
];

// Every step reads its state off real project data, so nothing here is a
// self-reported "have you read the docs" that could never tick. The agent step
// is the exception: copying the prompt is the only signal we get.
function completedSteps(data: OverviewProjectData, copied: Set<string>): Set<string> {
  const done = new Set(copied);
  if (data.project.apps.length > 0) {
    done.add("app");
  }
  if (data.keyspaces.length > 0) {
    done.add("keyspace");
  }
  if (data.ratelimits.length > 0) {
    done.add("ratelimit");
  }
  if (data.scenario !== "new") {
    done.add("team");
  }
  return done;
}

// Collapsed height: one row of content plus a sliver of the next, so the fade
// says "there is more" instead of cutting into a row you were reading.
const COLLAPSED_ROWS_PX = 106;
// Cuts partway through the first row's hint line. Landing on a row boundary
// leaves the gradient with nothing to act on, so the card just looks like it
// ends; biting into the text is what makes it read as clipped.
const COLLAPSED_GRID_PX = 46;

function GettingStarted({
  data,
  onDismiss,
  columns = 1,
}: {
  data: OverviewProjectData;
  onDismiss: () => void;
  /** Wide layouts wrap the steps into columns instead of one tall stack. */
  columns?: 1 | 3;
}) {
  const [copied, setCopied] = useState<Set<string>>(new Set());
  const [open, setOpen] = useState(false);
  const done = completedSteps(data, copied);
  const scope = { workspaceSlug: data.workspaceSlug, projectId: data.project.id };
  const pct = (done.size / STEPS.length) * 100;
  const hidden = STEPS.length - (columns === 3 ? columns : 2);

  // Numbers count what's left, not the original position — otherwise a
  // half-finished list reads "1 ✓ ✓ 4 ✓" with holes in it.
  let step = 0;

  return (
    <div className="group relative">
      <Card>
        <div className="px-3.5 pt-3 pb-2.5">
          <div className="flex items-center justify-between gap-2">
            <span className="text-[13px] font-medium text-accent-12">Getting started</span>
            <span className="shrink-0 text-xs tabular-nums text-gray-9">
              {done.size}/{STEPS.length}
            </span>
          </div>
          <div className="mt-2.5 h-1.5 overflow-hidden rounded-full bg-gray-3">
            <div className="h-full rounded-full bg-accent-12" style={{ width: `${pct}%` }} />
          </div>
        </div>
        <div
          className="relative overflow-hidden rounded-b-lg"
          style={
            open ? undefined : { maxHeight: columns === 3 ? COLLAPSED_GRID_PX : COLLAPSED_ROWS_PX }
          }
        >
          <div
            className={cn(
              columns === 3
                ? "grid grid-cols-1 gap-x-4 px-2 pb-2 sm:grid-cols-2 xl:grid-cols-3"
                : "divide-y divide-grayA-4",
            )}
          >
            {STEPS.map((item) => {
              const isDone = done.has(item.key);
              if (!isDone) {
                step += 1;
              }
              const body = (
                <>
                  <span
                    className={cn(
                      "flex size-5 shrink-0 items-center justify-center rounded-full",
                      isDone
                        ? "bg-success-3 text-success-11"
                        : "bg-gray-3 text-[11px] font-medium text-gray-9",
                    )}
                  >
                    {isDone ? <Check iconSize="sm-regular" /> : step}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div
                      className={cn(
                        "truncate text-[13px] leading-4",
                        isDone ? "text-gray-9" : "text-accent-12",
                      )}
                    >
                      {item.label}
                    </div>
                    <div className="mt-0.5 truncate text-xs text-gray-9">
                      {isDone ? item.done(data) : item.hint}
                    </div>
                  </div>
                </>
              );
              const rowClass = cn(
                "flex w-full items-center gap-2.5 px-3.5 py-3 text-left transition-colors hover:bg-grayA-2",
                columns === 3 && "rounded-md",
              );

              if (item.copy) {
                return (
                  <button
                    key={item.key}
                    type="button"
                    className={rowClass}
                    onClick={() => {
                      navigator.clipboard?.writeText(item.copy ?? "");
                      setCopied((prev) => new Set(prev).add(item.key));
                    }}
                  >
                    {body}
                  </button>
                );
              }
              return (
                <Link
                  key={item.key}
                  href={item.href?.(scope) ?? ("#" as Route)}
                  className={rowClass}
                >
                  {body}
                </Link>
              );
            })}
          </div>
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-background via-background/85 to-transparent transition-opacity duration-150 ease-out",
              // Shorter mask on the wide grid: at 62px of visible content an
              // h-12 gradient washes out the labels it is meant to reveal.
              columns === 3 ? "h-6" : "h-12",
              open ? "opacity-0" : "opacity-100",
            )}
          />
        </div>
      </Card>
      {/* Vercel's disclosure: the chevron straddles the bottom edge so the card
          reads as clipped rather than finished. */}
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        aria-expanded={open}
        aria-label={open ? "Collapse steps" : `Show ${hidden} more steps`}
        className="absolute -bottom-3.5 left-1/2 flex size-7 -translate-x-1/2 items-center justify-center rounded-full border border-grayA-4 bg-background text-gray-11 shadow-sm transition-colors hover:text-accent-12"
      >
        <span
          className={cn("block transition-transform duration-150 ease-out", open && "rotate-180")}
        >
          <ChevronDown iconSize="sm-regular" />
        </span>
      </button>
      {/* Same treatment as the agent setup card: the dismiss sits off the corner
          and only appears on hover, so it never competes with the progress. */}
      <button
        type="button"
        onClick={onDismiss}
        aria-label="Dismiss"
        className="absolute -right-1.5 -top-1.5 hidden size-5 items-center justify-center rounded-full border border-grayA-4 bg-background text-gray-9 shadow-sm hover:text-accent-12 group-hover:flex"
      >
        <XMark iconSize="sm-regular" />
      </button>
    </div>
  );
}

/* ---------------------------------------------------------------- help card */

// Outlives the checklist on purpose: that can be dismissed for good, so without
// this the rail would collapse to nothing.
const HELP_LINKS = [
  { href: DISCORD_URL, label: "Ask in Discord", sub: "Community answers", icon: Discord },
  { href: SUPPORT_URL, label: "Get support", sub: "Talk to an engineer", icon: Chats },
  { href: TEMPLATES_URL, label: "Templates", sub: "Start from an example", icon: Layers2 },
  { href: DOCS_URL, label: "Documentation", sub: "Guides and API reference", icon: BookBookmark },
];

// The borderless card header only works above full-bleed stacked rows, so the
// wide version drops the card and becomes a labelled tile row instead.
function HelpCard({ wide = false }: { wide?: boolean }) {
  if (wide) {
    return (
      <div>
        <span className="text-sm font-medium text-accent-12">Need help?</span>
        <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
          {HELP_LINKS.map((link) => (
            <a
              key={link.label}
              href={link.href}
              target="_blank"
              rel="noreferrer"
              className="flex flex-col gap-2 rounded-lg border border-grayA-4 p-3 transition-colors hover:border-grayA-7"
            >
              <link.icon iconSize="lg-regular" className="text-gray-9" />
              <div className="min-w-0">
                <div className="text-[13px] font-medium text-accent-12">{link.label}</div>
                <div className="mt-0.5 truncate text-xs text-gray-9">{link.sub}</div>
              </div>
            </a>
          ))}
        </div>
      </div>
    );
  }
  return (
    <Card className="overflow-hidden">
      <div className="px-3.5 pt-3 pb-1.5">
        <span className="text-[13px] font-medium text-accent-12">Need help?</span>
      </div>
      <div className="divide-y divide-grayA-4">
        {HELP_LINKS.map((link) => (
          <HelpRow key={link.label} href={link.href} icon={<link.icon iconSize="md-regular" />}>
            {link.label}
          </HelpRow>
        ))}
      </div>
    </Card>
  );
}

function HelpRow({
  href,
  icon,
  children,
}: {
  href: string;
  icon: ReactNode;
  children: ReactNode;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="flex items-center gap-2.5 px-3.5 py-3 transition-colors hover:bg-grayA-2"
    >
      <span className="shrink-0 text-gray-9">{icon}</span>
      <span className="text-[13px] text-gray-11">{children}</span>
    </a>
  );
}

/* ---------------------------------------------------------------- apps list */

function AppsCard({
  appsHref,
  newAppHref,
  deployments,
  hasApps,
  keyspaceCount,
  ratelimitCount,
}: {
  appsHref: Route;
  newAppHref: Route;
  deployments: DeploymentMock[];
  hasApps: boolean;
  keyspaceCount: number;
  ratelimitCount: number;
}) {
  // One row per app: its latest deployment carries the commit, branch, actor and
  // status, so the row says what shipped rather than just naming the app.
  const latestPerApp = new Map<string, DeploymentMock>();
  for (const deployment of deployments) {
    const current = latestPerApp.get(deployment.appId);
    if (!current || deployment.timeAgoMin < current.timeAgoMin) {
      latestPerApp.set(deployment.appId, deployment);
    }
  }
  const rows = [...latestPerApp.values()].sort((a, b) => a.timeAgoMin - b.timeAgoMin);

  return (
    <Card className="overflow-hidden">
      <div className="flex items-center justify-between gap-3 px-4 pt-3 pb-1.5">
        <div className="flex items-center gap-2">
          <Link href={appsHref} className="text-[13px] font-medium text-accent-12 hover:underline">
            Apps
          </Link>
          {rows.length > 0 && (
            <span className="rounded-full bg-grayA-3 px-1.5 py-0.5 text-[11px] font-medium tabular-nums text-gray-11">
              {rows.length}
            </span>
          )}
        </div>
        <Button size="sm" variant="outline" render={<Link href={newAppHref} />}>
          <Plus iconSize="sm-regular" />
          Create app
        </Button>
      </div>
      {rows.length > 0 ? (
        <div className="divide-y divide-grayA-4">
          {rows.map((deployment) => (
            <AppRow key={deployment.appId} deployment={deployment} />
          ))}
        </div>
      ) : (
        <AppsEmpty
          newAppHref={newAppHref}
          hasApps={hasApps}
          keyspaceCount={keyspaceCount}
          ratelimitCount={ratelimitCount}
        />
      )}
    </Card>
  );
}

function AppRow({ deployment }: { deployment: DeploymentMock }) {
  const appHomeHref = useAppHomeHref();
  const scope = useProjectScope();
  const href = appHomeHref({ ...scope, appId: deployment.appId });

  // Row shell copied from the deployments list: a full-bleed absolute link under
  // the content, and every interactive child lifted to z-20 so its popover or
  // tooltip isn't swallowed by that overlay.
  return (
    <div className={cn(APP_GRID, "relative px-4 py-2.5 transition-colors hover:bg-grayA-2")}>
      <Link
        href={href}
        className="absolute inset-0 z-10"
        aria-label={`View ${deployment.appName}`}
      />
      <span className="flex min-w-0 items-center gap-2.5">
        <span className="shrink-0 text-gray-12">
          {deployment.appSource === "github" ? (
            <Github iconSize="xl-medium" />
          ) : (
            <Terminal iconSize="xl-medium" />
          )}
        </span>
        <span className="shrink-0 text-[13px] font-medium text-accent-12">
          {deployment.appName}
        </span>
        <span className="truncate text-[13px] text-gray-9" title={deployment.message}>
          {deployment.message}
        </span>
      </span>
      <span className="flex min-w-0 items-center gap-2">
        <CodeBranch iconSize="sm-regular" className="shrink-0 text-accent-12" />
        <span
          className="truncate font-mono text-xs leading-4 text-accent-12"
          title={deployment.branch}
        >
          {deployment.branch}
        </span>
      </span>
      <DeploymentStatusBadge status={deployment.status} />
      <span className="justify-self-end whitespace-nowrap text-xs tabular-nums text-gray-9">
        {fmtTimeAgo(deployment.timeAgoMin)}
      </span>
      <span className="relative z-20 flex justify-end">
        <AppActions projectId={scope.projectId} appId={deployment.appId}>
          <Button variant="ghost" size="icon" className="shrink-0" title="App actions">
            <Dots iconSize="sm-regular" />
          </Button>
        </AppActions>
      </span>
    </div>
  );
}

function AppsEmpty({
  newAppHref,
  hasApps,
  keyspaceCount,
  ratelimitCount,
}: {
  newAppHref: Route;
  hasApps: boolean;
  keyspaceCount: number;
  ratelimitCount: number;
}) {
  // A migrated project already verifies keys, so the empty state says what is
  // missing (nothing served) instead of implying the project is unused.
  const migrated = !hasApps && (keyspaceCount > 0 || ratelimitCount > 0);
  return (
    <div className="mx-4 mb-4 flex flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-grayA-4 px-4 py-10 text-center">
      <Cube iconSize="xl-thin" className="text-gray-9" />
      <div className="mt-3 text-[13px] font-medium text-accent-12">No apps yet</div>
      <p className="max-w-xs text-[13px] text-gray-9">
        {migrated
          ? "Your keys are being verified, but nothing is served yet."
          : "Deploy from a repo or from the CLI and your apps land here."}
      </p>
      <div className="mt-4 flex shrink-0 items-center gap-2">
        <Button size="md" variant="primary" render={<Link href={newAppHref} />}>
          <Github iconSize="sm-regular" />
          Deploy an app
        </Button>
        <Button size="md" variant="outline" render={<Link href={newAppHref} />}>
          <Terminal iconSize="sm-regular" />
          Deploy from CLI
        </Button>
      </div>
    </div>
  );
}
