"use client";

import { AppActions } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/_components/apps-list/app-actions";
import { DeploymentStatusBadge } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/deployment-status-badge";
import {
  AgentLogos,
  PROMPT,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/agent-setup";
import type { Mark } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/marks";
import {
  RailListShell,
  RailRow,
  type RowItem,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/rail";
import type { RowVariant } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/scenario";
import {
  keyspaceSeries,
  ratelimitSeries,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/series";
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

const DISCORD_URL = "https://unkey.com/discord";
const SUPPORT_URL = "https://unkey.com/support";
const DOCS_URL = "https://unkey.com/docs";

// One template for the header and the rows — the whole point of the table
// treatment is that columns line up, so it can only be declared once.
const APP_GRID = "grid grid-cols-[minmax(0,1fr)_120px_104px_76px_32px] items-center gap-3";

// Matches the projects rail: divided rows with a bar mark, and the same
// valid/success bar color the api and ratelimit list pages use.
const ROW_VARIANT: RowVariant = "list";
// Full-width cards have room for the 24h volume, so resource rows use the metric
// treatment rather than the rail's compact list row.
const ROW_METRIC_VARIANT: RowVariant = "metric";
const ROW_MARK: Mark = "bars";
// Series colours per data type: keyspaces read as verifications, ratelimits as
// checks, both following the chart scheme.
const KEYSPACE_OK = "hsl(var(--chart-verify-ok))";
const KEYSPACE_BAD = "hsl(var(--chart-verify-bad))";
const RATELIMIT_OK = "hsl(var(--chart-limit-ok))";
const RATELIMIT_BAD = "hsl(var(--chart-limit-bad))";

// The overview shows the most recent handful; the apps page is where the full
// list lives, so a footer link goes there rather than growing this card.
const MAX_APP_ROWS = 5;

// Two rows of content plus a sliver of the third, so the fade says "there is
// more" instead of cutting into a row you were reading.
const COLLAPSED_ROWS_PX = 132;

function useProjectScope(): { workspaceSlug: string; projectId: string } {
  const workspace = useWorkspaceNavigation();
  const params = useParams<{ projectId: string }>();
  return { workspaceSlug: workspace.slug, projectId: params.projectId };
}

// Apps are the page. No resource-count strip on top: the apps section already
// says how many apps there are, and the rail lists the actual keyspaces and
// ratelimits rather than counting them.
export function ProjectOverview({ data }: { data: OverviewProjectData }) {
  const { workspaceSlug, project, keyspaces, ratelimits, deployments } = data;
  const [dismissed, setDismissed] = useState(false);
  const scope = { workspaceSlug, projectId: project.id };

  const apps = (
    <AppsCard
      appsHref={routes.projects.apps.list(scope)}
      newAppHref={routes.projects.apps.new(scope)}
      deployments={deployments}
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
    />
  );

  // The checklist shares the resource row rather than leading the page, so the
  // column count follows how many cards actually have something to show.
  const cardCount =
    (keyspaces.length > 0 ? 1 : 0) + (ratelimits.length > 0 ? 1 : 0) + (dismissed ? 0 : 1);

  return (
    <div className="flex flex-col gap-6">
      {apps}
      <div
        className={cn(
          "grid grid-cols-1 gap-5",
          cardCount >= 3 ? "lg:grid-cols-3" : cardCount === 2 ? "lg:grid-cols-2" : "",
        )}
      >
        {!dismissed && <GettingStarted data={data} onDismiss={() => setDismissed(true)} />}
        {resources}
      </div>
      <HelpCard wide />
    </div>
  );
}

/* ----------------------------------------------------------- resource lists */

// One card per resource type with a row per keyspace / namespace, using the
// same shell and row the projects rail uses. No project name in the subtitle:
// on a project page it would repeat on every row.
//
// TODO: cap both cards at the 3 most active (by 24h volume) with a "View all"
// footer, matching AppsCard below. The projects rail needs the same cap — see
// Rail in projects/_components/prototype/rail.tsx.
function ResourceLists({
  workspaceSlug,
  projectId,
  keyspaces,
  ratelimits,
}: {
  workspaceSlug: string;
  projectId: string;
  keyspaces: OverviewProjectData["keyspaces"];
  ratelimits: OverviewProjectData["ratelimits"];
}) {
  const keyspaceRows: RowItem[] = keyspaces.map((ks) => {
    const series = keyspaceSeries(ks);
    return {
      id: ks.id,
      title: ks.name,
      subtitle: `${fmtInt(ks.keyCount)} keys`,
      value: fmtCompact(series.total),
      spark: series.totals,
      buckets: series.buckets,
      errorRatio: (100 - ks.validPct) / 100,
      stroke: KEYSPACE_OK,
      errorStroke: KEYSPACE_BAD,
      labels: { ok: "valid", bad: "invalid" },
      kind: "keyspace",
      href: routes.apis.detail({ workspaceSlug, apiId: ks.id }),
    };
  });
  const ratelimitRows: RowItem[] = ratelimits.map((rl) => {
    const series = ratelimitSeries(rl);
    return {
      id: rl.id,
      title: rl.name,
      subtitle: `${rl.blockedPct}% blocked`,
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
    <>
      {keyspaceRows.length > 0 && (
        <RailListShell
          title="Keyspaces"
          variant={ROW_VARIANT}
          subtitle="24h"
          viewAllHref={routes.projects.keyspaces({ workspaceSlug, projectId })}
        >
          {keyspaceRows.map((item) => (
            <RailRow key={item.id} item={item} variant={ROW_METRIC_VARIANT} mark={ROW_MARK} />
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
            <RailRow key={item.id} item={item} variant={ROW_METRIC_VARIANT} mark={ROW_MARK} />
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
  done: (data: OverviewProjectData, deployments: DeploymentMock[]) => string;
  href?: (scope: { workspaceSlug: string; projectId: string }) => Route;
};

// The path to a deployed project, in order. Repo and deploy stay separate on
// purpose: a repo can be connected while every build fails, and that gap is
// exactly what someone stuck needs to see.
const STEPS: Step[] = [
  {
    key: "github",
    label: "Install the GitHub app",
    hint: "Let Unkey read your repositories",
    done: () => "Installed",
    href: (scope) => routes.projects.apps.new(scope),
  },
  {
    key: "repo",
    label: "Connect a repo",
    hint: "Point an app at a branch",
    done: ({ project }) => {
      const connected = project.apps.filter((app) => app.source === "github");
      return connected.length === 1
        ? `unkey/${connected[0].name}`
        : `${fmtInt(connected.length)} repos connected`;
    },
    href: (scope) => routes.projects.apps.new(scope),
  },
  {
    key: "deploy",
    label: "Deploy",
    hint: "Ship it to production",
    // The commit message says what shipped; a sha says only that something did.
    done: (_data, deployments) => {
      const ready = deployments
        .filter((d) => d.status === "ready")
        .sort((a, b) => a.timeAgoMin - b.timeAgoMin)[0];
      return ready?.message ?? "Live";
    },
    href: (scope) => routes.projects.apps.list(scope),
  },
  {
    key: "domain",
    label: "Add a custom domain",
    hint: "Serve on your own hostname",
    done: () => "Configured",
    href: (scope) => routes.projects.settings(scope),
  },
];

// A CLI-first project can never satisfy the two GitHub steps, and a checklist
// with permanently unreachable rows teaches people to ignore it — so they drop
// out entirely once the project has deployed without a repo.
function stepsFor(data: OverviewProjectData): Step[] {
  const hasApps = data.project.apps.length > 0;
  const anyRepo = data.project.apps.some((app) => app.source === "github");
  if (hasApps && !anyRepo) {
    return STEPS.filter((step) => step.key !== "github" && step.key !== "repo");
  }
  return STEPS;
}

// Every step reads off real project data — nothing here is self-reported, so no
// row can sit ticked while the thing it claims isn't true.
function completedSteps(data: OverviewProjectData, deployments: DeploymentMock[]): Set<string> {
  const done = new Set<string>();
  // Installing the app is workspace-level and the prototype store has no
  // installation record, so only the fully set-up scenario is treated as
  // having it — a migrated workspace had keys/ratelimits already but never
  // touched GitHub or deploy, so it can't be assumed here.
  if (data.scenario === "active") {
    done.add("github");
  }
  if (data.project.apps.some((app) => app.source === "github")) {
    done.add("repo");
  }
  if (deployments.some((d) => d.status === "ready")) {
    done.add("deploy");
  }
  // No domain field on the mocks yet, so this only ticks for the fully set-up
  // scenario rather than being derived.
  if (data.scenario === "active") {
    done.add("domain");
  }
  return done;
}

function GettingStarted({
  data,
  onDismiss,
  columns = 1,
  className,
}: {
  data: OverviewProjectData;
  onDismiss: () => void;
  /** Wide layouts wrap the steps into columns instead of one tall stack. */
  columns?: 1 | 3;
  className?: string;
}) {
  const steps = stepsFor(data);
  const done = completedSteps(data, data.deployments);
  const scope = { workspaceSlug: data.workspaceSlug, projectId: data.project.id };
  const allDone = steps.every((step) => done.has(step.key));
  const [expanded, setExpanded] = useState(false);
  const collapsible = columns === 1 && steps.length > 2;
  const collapsed = collapsible && !expanded;

  // Numbers count what's left, not the original position — otherwise a
  // half-finished list reads "1 ✓ ✓ 4 ✓" with holes in it.
  let step = 0;

  return (
    <div className={cn("group relative self-start", className)}>
      <Card className="flex flex-col">
        <div className="px-3.5 pt-3 pb-1.5">
          <span className="text-[13px] font-medium text-accent-12">Getting started</span>
        </div>
        <div
          className="relative overflow-hidden rounded-b-lg"
          style={collapsed ? { maxHeight: COLLAPSED_ROWS_PX } : undefined}
        >
          <div
            className={cn(
              columns === 3
                ? "grid grid-cols-1 gap-x-4 px-2 pb-2 sm:grid-cols-2 xl:grid-cols-3"
                : "divide-y divide-grayA-4",
            )}
          >
            {steps.map((item) => {
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
                      {isDone ? item.done(data, data.deployments) : item.hint}
                    </div>
                  </div>
                </>
              );
              const rowClass = cn(
                "flex w-full items-center gap-2.5 px-3.5 py-3 text-left transition-colors hover:bg-grayA-2",
                columns === 3 && "rounded-md",
              );

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
              "pointer-events-none absolute inset-x-0 bottom-0 h-9 bg-gradient-to-t from-background via-background/80 to-transparent transition-opacity duration-150 ease-out",
              collapsed ? "opacity-100" : "opacity-0",
            )}
          />
        </div>
      </Card>
      {/* Vercel's disclosure: the chevron straddles the bottom edge so the card
          reads as clipped rather than finished. */}
      {collapsible && (
        <button
          type="button"
          onClick={() => setExpanded((prev) => !prev)}
          aria-expanded={expanded}
          aria-label={expanded ? "Collapse steps" : `Show ${steps.length - 2} more steps`}
          className="absolute -bottom-3.5 left-1/2 flex size-7 -translate-x-1/2 items-center justify-center rounded-full border border-grayA-4 bg-background text-gray-11 shadow-sm transition-colors hover:text-accent-12"
        >
          <span
            className={cn(
              "block transition-transform duration-150 ease-out",
              expanded && "rotate-180",
            )}
          >
            <ChevronDown iconSize="sm-regular" />
          </span>
        </button>
      )}
      {/* Dismiss only appears once every step is done — while anything is
          outstanding the card is the nudge, so there's nothing to dismiss. */}
      {allDone && (
        <button
          type="button"
          onClick={onDismiss}
          aria-label="Dismiss"
          className="absolute -right-1.5 -top-1.5 hidden size-5 items-center justify-center rounded-full border border-grayA-4 bg-background text-gray-9 shadow-sm hover:text-accent-12 group-hover:flex"
        >
          <XMark iconSize="sm-regular" />
        </button>
      )}
    </div>
  );
}

/* ---------------------------------------------------------------- help card */

// Outlives the checklist on purpose: that can be dismissed for good, so without
// this the rail would collapse to nothing.
const HELP_LINKS = [
  { href: DISCORD_URL, label: "Ask in Discord", sub: "Community answers", icon: Discord },
  { href: SUPPORT_URL, label: "Get support", sub: "Talk to an engineer", icon: Chats },
  { href: DOCS_URL, label: "Documentation", sub: "Guides and API reference", icon: BookBookmark },
];

// The borderless card header only works above full-bleed stacked rows, so the
// wide version drops the card and becomes a labelled tile row instead. The
// agent tile sits where a link would, but copies the setup prompt instead of
// navigating — same interaction as the rail's AgentSetup card.
function HelpCard({ wide = false }: { wide?: boolean }) {
  if (wide) {
    return (
      <div>
        <span className="text-sm font-medium text-accent-12">Need help?</span>
        <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <AgentHelpTile />
          {HELP_LINKS.map((link) => (
            <HelpTile key={link.label} {...link} />
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
        <AgentHelpRow />
        {HELP_LINKS.map((link) => (
          <HelpRow key={link.label} href={link.href} icon={<link.icon iconSize="md-regular" />}>
            {link.label}
          </HelpRow>
        ))}
      </div>
    </Card>
  );
}

function HelpTile({
  href,
  icon: Icon,
  label,
  sub,
}: {
  href: string;
  icon: typeof BookBookmark;
  label: string;
  sub: string;
}) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="flex flex-col gap-2 rounded-lg border border-grayA-4 p-3 transition-colors hover:border-grayA-7"
    >
      <Icon iconSize="lg-regular" className="text-gray-9" />
      <div className="min-w-0">
        <div className="text-[13px] font-medium text-accent-12">{label}</div>
        <div className="mt-0.5 truncate text-xs text-gray-9">{sub}</div>
      </div>
    </a>
  );
}

function useCopyPrompt() {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(PROMPT);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1800);
  };
  return { copied, copy };
}

function AgentHelpTile() {
  const { copied, copy } = useCopyPrompt();
  return (
    <button
      type="button"
      onClick={copy}
      className="flex flex-col gap-2 rounded-lg border border-grayA-4 p-3 text-left transition-colors hover:border-grayA-7"
    >
      <AgentLogos />
      <div className="min-w-0">
        <div className="text-[13px] font-medium text-accent-12">
          {copied ? "Copied" : "Set up with your agent"}
        </div>
        <div className="mt-0.5 truncate text-xs text-gray-9">Claude, Cursor, Codex</div>
      </div>
    </button>
  );
}

function AgentHelpRow() {
  const { copied, copy } = useCopyPrompt();
  return (
    <button
      type="button"
      onClick={copy}
      className="flex w-full items-center gap-2.5 px-3.5 py-3 text-left transition-colors hover:bg-grayA-2"
    >
      <AgentLogos />
      <span className="min-w-0 flex-1 truncate text-[13px] text-gray-11">
        {copied ? "Copied" : "Set up with your agent"}
      </span>
    </button>
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
}: {
  appsHref: Route;
  newAppHref: Route;
  deployments: DeploymentMock[];
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
  // Most recently deployed first, so the list answers "what just shipped".
  const rows = [...latestPerApp.values()].sort((a, b) => a.timeAgoMin - b.timeAgoMin);
  const visible = rows.slice(0, MAX_APP_ROWS);
  const overflow = rows.length - visible.length;

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
        <>
          <div className="divide-y divide-grayA-4">
            {visible.map((deployment) => (
              <AppRow key={deployment.appId} deployment={deployment} />
            ))}
          </div>
          {overflow > 0 && (
            <div className="p-1.5">
              <Button
                size="sm"
                variant="ghost"
                className="w-full text-gray-11 hover:text-accent-12"
                render={<Link href={appsHref} />}
              >
                View all
              </Button>
            </div>
          )}
        </>
      ) : (
        <AppsEmpty newAppHref={newAppHref} />
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
        <span className="ml-1 truncate text-[13px] text-accent-12" title={deployment.message}>
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

function AppsEmpty({ newAppHref }: { newAppHref: Route }) {
  return (
    <div className="mx-4 mb-4 flex flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-grayA-4 px-4 py-10 text-center">
      <Cube iconSize="xl-thin" className="text-gray-9" />
      <div className="mt-3 text-[13px] font-medium text-accent-12">No apps yet</div>
      <p className="max-w-xs text-[13px] text-gray-9">
        Deploy from a repo or the CLI to add your first app.
      </p>
      <div className="mt-4 flex shrink-0 items-center gap-2">
        <Button size="md" variant="outline" render={<Link href={newAppHref} />}>
          <Github iconSize="sm-regular" />
          Deploy from GitHub
        </Button>
        <Button size="md" variant="outline" render={<Link href={newAppHref} />}>
          <Terminal iconSize="sm-regular" />
          Deploy from CLI
        </Button>
      </div>
    </div>
  );
}
