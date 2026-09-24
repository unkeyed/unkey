"use client";

import {
  DEPLOYMENT_STATUS_LABELS,
  type DeploymentStatusGroup,
  deploymentStatusColor,
  statusGroupOf,
} from "@/lib/collections/deploy/deployment-status";
import { trpc } from "@/lib/trpc/client";
import type { OverviewApp, ProjectOverview } from "@/lib/trpc/routers/deploy/project/overview";
import {
  Github,
  IconCodeBranchOutline18,
  IconCubeOutline18,
  IconEarthOutline18,
  IconGaugeOutline18,
  IconLayers2Outline18,
  IconNodesOutline18,
  IconPlusOutline18,
  IconShieldKeyOutline18,
  IconSquareTerminalOutline18,
  IconTerminalOutline18,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { type OverviewModel, ago, compact } from "./overview-model";

const WINDOW_MS = 7 * 24 * 60 * 60 * 1000;
const MAX_APPS = 5;

const APP_ORDER: Record<DeploymentStatusGroup, number> = {
  failed: 0,
  blocked: 1,
  building: 2,
  ready: 3,
  queued: 4,
  stopped: 5,
  cancelled: 6,
  skipped: 6,
  superseded: 6,
};

function appRank(app: OverviewApp): number {
  return app.latest ? APP_ORDER[statusGroupOf(app.latest.status)] : 7;
}

export type CanvasLinks = {
  app: (appId: string) => Route;
  allApps: Route;
  keyspace: (apiId: string) => Route;
  allKeyspaces: Route;
  ratelimit: (namespaceId: string) => Route;
  allRatelimits: Route;
  logs: Route;
};

export type CanvasActions = {
  createApp: () => void;
  createKeyspace: () => void;
  createRatelimit: () => void;
};

export function Canvas({
  data,
  model,
  links,
  actions,
}: {
  data: ProjectOverview;
  model: OverviewModel;
  links: CanvasLinks;
  actions: CanvasActions;
}) {
  const hasApps = data.apps.length > 0;
  const hasKeyspaces = data.keyspaces.length > 0;
  const hasRatelimits = data.ratelimits.length > 0;

  return (
    <div
      className="relative w-full overflow-x-auto rounded-xl border border-border bg-background"
      style={{
        backgroundImage: "radial-gradient(var(--color-grayA-5) 1px, transparent 1px)",
        backgroundSize: "14px 14px",
      }}
    >
      <div className="flex min-h-[420px] w-full min-w-[860px] items-start justify-center px-6 pt-6 pb-24">
        {hasApps ? (
          <>
            <Group label="Network">
              <EdgeCard data={data} />
            </Group>
            <Connector />
            <Group label={`Apps · ${data.apps.length}`} href={links.allApps}>
              {[...data.apps]
                .sort((a, b) => appRank(a) - appRank(b))
                .slice(0, MAX_APPS)
                .map((app) => (
                  <AppCard key={app.id} app={app} href={links.app(app.id)} />
                ))}
              {data.apps.length > MAX_APPS && (
                <Link
                  href={links.allApps}
                  className="rounded-lg border border-dashed border-border px-3 py-2 text-center text-xs text-gray-11 hover:text-gray-12"
                >
                  +{data.apps.length - MAX_APPS} more apps
                </Link>
              )}
            </Group>
          </>
        ) : (
          <Group label="Apps">
            <GhostCard
              icon={<IconCubeOutline18 />}
              title="Deploy an app"
              description={
                model.shape === "api"
                  ? "Run the service behind your keys on Unkey, next to them."
                  : "From a GitHub repo or a container image."
              }
              onClick={actions.createApp}
            />
          </Group>
        )}

        <Connector dashed={!hasKeyspaces && !hasRatelimits} />

        <Group label="API">
          {hasKeyspaces ? (
            <KeyspacesCard data={data} links={links} />
          ) : (
            <GhostCard
              icon={<IconNodesOutline18 />}
              title="Protect it with keys"
              description="Issue and verify API keys for your users."
              onClick={actions.createKeyspace}
            />
          )}
          {hasRatelimits ? (
            <RatelimitsCard data={data} links={links} />
          ) : (
            <GhostCard
              icon={<IconGaugeOutline18 />}
              title="Add a ratelimit"
              description="Cap requests per user, key, or IP."
              onClick={actions.createRatelimit}
            />
          )}
        </Group>
      </div>

      {model.shape === "empty" && <AgentHint />}
      <CommandBar actions={actions} />
    </div>
  );
}

function Group({ label, href, children }: { label: string; href?: Route; children: ReactNode }) {
  return (
    <div className="flex min-w-0 max-w-[340px] flex-1 flex-col gap-2 rounded-xl bg-grayA-2 p-2.5">
      <div className="flex items-center justify-between px-1 pb-0.5">
        <span className="text-xs text-gray-11">{label}</span>
        {href && (
          <Link href={href} className="text-xs text-gray-9 hover:text-gray-12">
            View all
          </Link>
        )}
      </div>
      {children}
    </div>
  );
}

function Connector({ dashed = false }: { dashed?: boolean }) {
  return (
    <div
      className={cn(
        "mt-[58px] w-8 shrink-0 border-t",
        dashed ? "border-dashed border-grayA-6" : "border-grayA-7",
      )}
    />
  );
}

function Card({ children, href }: { children: ReactNode; href?: Route }) {
  const cls =
    "block rounded-lg border border-border bg-raised shadow-xs transition-colors hover:border-strong";
  return href ? (
    <Link href={href} className={cls}>
      {children}
    </Link>
  ) : (
    <div className={cls}>{children}</div>
  );
}

function CardHeader({ icon, title, right }: { icon: ReactNode; title: string; right?: ReactNode }) {
  return (
    <div className="flex items-center gap-2 border-b border-border px-3 py-2.5">
      <span className="text-gray-11 [&_svg]:size-4">{icon}</span>
      <span className="min-w-0 truncate text-[13px] font-medium text-gray-12">{title}</span>
      {right && <span className="ml-auto shrink-0">{right}</span>}
    </div>
  );
}

function Row({ icon, label, value }: { icon?: ReactNode; label: string; value: ReactNode }) {
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 text-xs">
      {icon && <span className="text-gray-9 [&_svg]:size-3.5">{icon}</span>}
      <span className="min-w-0 truncate text-gray-11">{label}</span>
      <span className="ml-auto shrink-0 text-gray-12">{value}</span>
    </div>
  );
}

function Enabled() {
  return (
    <span className="inline-flex items-center gap-1.5 text-success-11">
      <span className="size-1.5 rounded-full bg-success-9" />
      Enabled
    </span>
  );
}

function EdgeCard({ data }: { data: ProjectOverview }) {
  const domains = data.apps.filter((a) => a.domain).length;
  return (
    <Card>
      <CardHeader icon={<IconEarthOutline18 />} title="Unkey gateway" />
      <div className="py-1">
        <Row icon={<IconShieldKeyOutline18 />} label="TLS" value={<Enabled />} />
        <Row icon={<IconEarthOutline18 />} label="Domains" value={domains || "None yet"} />
        <Row
          icon={<IconNodesOutline18 />}
          label="Key verification"
          value={data.keyspaces.length ? <Enabled /> : <span className="text-gray-9">Off</span>}
        />
      </div>
    </Card>
  );
}

function AppCard({ app, href }: { app: OverviewApp; href: Route }) {
  const d = app.latest;
  const icon =
    app.sourceType === "oci" ? (
      <IconLayers2Outline18 />
    ) : app.repositoryFullName ? (
      <Github />
    ) : (
      <IconTerminalOutline18 />
    );
  return (
    <Card href={href}>
      <CardHeader
        icon={icon}
        title={app.name}
        right={
          d ? (
            <span className="inline-flex items-center gap-1.5 text-xs text-gray-11">
              <span className={cn("size-1.5 rounded-full", deploymentStatusColor(d.status))} />
              {DEPLOYMENT_STATUS_LABELS[d.status]}
            </span>
          ) : (
            <span className="text-xs text-gray-9">Not deployed</span>
          )
        }
      />
      {d ? (
        <div className="flex items-center gap-2 px-3 py-2 text-xs">
          <IconCodeBranchOutline18 className="size-3.5 shrink-0 text-gray-9" />
          <span className="max-w-24 shrink-0 truncate font-mono text-gray-11">
            {d.branch ?? "main"}
          </span>
          <span className="min-w-0 flex-1 truncate text-gray-11">
            {d.commitMessage?.split("\n")[0] ?? app.domain ?? "Image deployment"}
          </span>
          <span className="shrink-0 text-gray-9">{ago(d.createdAt)}</span>
        </div>
      ) : (
        <div className="px-3 py-2 text-xs text-gray-9">Push to deploy, or deploy from here.</div>
      )}
    </Card>
  );
}

function KeyspacesCard({ data, links }: { data: ProjectOverview; links: CanvasLinks }) {
  return (
    <Card>
      <CardHeader
        icon={<IconNodesOutline18 />}
        title="Keyspaces"
        right={
          <Link href={links.allKeyspaces} className="text-xs text-gray-9 hover:text-gray-12">
            {data.keyspaces.length}
          </Link>
        }
      />
      <div className="py-1">
        {data.keyspaces.slice(0, 3).map((ks) => (
          <KeyspaceRow
            key={ks.apiId}
            name={ks.name}
            keyAuthId={ks.keyAuthId}
            keyCount={ks.keyCount}
            href={links.keyspace(ks.apiId)}
          />
        ))}
      </div>
    </Card>
  );
}

function useWindow() {
  return useMemo(() => {
    const endTime = Date.now();
    return { startTime: endTime - WINDOW_MS, endTime };
  }, []);
}

function KeyspaceRow({
  name,
  keyAuthId,
  keyCount,
  href,
}: {
  name: string;
  keyAuthId: string;
  keyCount: number;
  href: Route;
}) {
  const window = useWindow();
  const { data } = trpc.api.overview.timeseries.useQuery(
    { keyspaceId: keyAuthId, ...window, since: "" },
    { trpc: { context: { skipBatch: true } } },
  );
  const points = data?.timeseries?.map((p) => p.y.total) ?? [];
  const total = points.reduce((a, b) => a + b, 0);
  return (
    <Link href={href} className="flex items-center gap-3 px-3 py-1.5 hover:bg-grayA-2">
      <div className="min-w-0 flex-1">
        <div className="truncate text-xs text-gray-12">{name}</div>
        <div className="text-[11px] text-gray-9">
          {compact(keyCount)} keys · {compact(total)} verifications
        </div>
      </div>
      <Spark points={points} className="bg-success-9" />
    </Link>
  );
}

function RatelimitsCard({ data, links }: { data: ProjectOverview; links: CanvasLinks }) {
  const window = useWindow();
  const shown = data.ratelimits.slice(0, 3);
  const { data: ts } = trpc.ratelimit.logs.queryRatelimitTimeseriesBatch.useQuery({
    namespaceIds: shown.map((n) => n.id),
    ...window,
  });
  return (
    <Card>
      <CardHeader
        icon={<IconGaugeOutline18 />}
        title="Ratelimits"
        right={
          <Link href={links.allRatelimits} className="text-xs text-gray-9 hover:text-gray-12">
            {data.ratelimits.length}
          </Link>
        }
      />
      <div className="py-1">
        {shown.map((ns) => {
          const series = ts?.timeseriesByNamespace[ns.id] ?? [];
          const total = series.reduce((a, p) => a + p.y.total, 0);
          const passed = series.reduce((a, p) => a + p.y.passed, 0);
          const blocked = total ? ((total - passed) / total) * 100 : 0;
          return (
            <Link
              key={ns.id}
              href={links.ratelimit(ns.id)}
              className="flex items-center gap-3 px-3 py-1.5 hover:bg-grayA-2"
            >
              <div className="min-w-0 flex-1">
                <div className="truncate text-xs text-gray-12">{ns.name}</div>
                <div className="text-[11px] text-gray-9">
                  {compact(total)} requests
                  {total > 0 && (
                    <span className={cn(blocked > 5 && "text-warning-11")}>
                      {" "}
                      · {blocked.toFixed(1)}% blocked
                    </span>
                  )}
                </div>
              </div>
              <Spark points={series.map((p) => p.y.total)} className="bg-info-9" />
            </Link>
          );
        })}
      </div>
    </Card>
  );
}

function Spark({ points, className }: { points: number[]; className: string }) {
  const tail = points.slice(-24);
  const max = Math.max(1, ...tail);
  if (tail.every((p) => p === 0)) {
    return <span className="text-[11px] text-gray-8">No traffic</span>;
  }
  return (
    <div className="flex h-5 w-16 shrink-0 items-end gap-px">
      {tail.map((p, i) => (
        <span
          // biome-ignore lint/suspicious/noArrayIndexKey: positional bars
          key={i}
          className={cn("flex-1 rounded-[1px]", p ? className : "bg-grayA-4")}
          style={{ height: `${Math.max(8, (p / max) * 100)}%` }}
        />
      ))}
    </div>
  );
}

function GhostCard({
  icon,
  title,
  description,
  onClick,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group flex items-start gap-2.5 rounded-lg border border-dashed border-grayA-6 bg-raised/60 px-3 py-3 text-left transition-colors hover:border-strong hover:bg-raised"
    >
      <span className="mt-0.5 text-gray-9 group-hover:text-gray-12 [&_svg]:size-4">{icon}</span>
      <span className="min-w-0">
        <span className="block text-[13px] font-medium text-gray-12">{title}</span>
        <span className="block text-xs text-gray-9">{description}</span>
      </span>
      <IconPlusOutline18 className="ml-auto mt-0.5 size-3.5 text-gray-9 group-hover:text-gray-12" />
    </button>
  );
}

function CommandBar({ actions }: { actions: CanvasActions }) {
  const items = [
    { label: "App", icon: <IconCubeOutline18 />, run: actions.createApp },
    { label: "Keyspace", icon: <IconNodesOutline18 />, run: actions.createKeyspace },
    { label: "Ratelimit", icon: <IconGaugeOutline18 />, run: actions.createRatelimit },
  ];
  return (
    <div className="absolute bottom-5 left-1/2 flex -translate-x-1/2 items-center gap-0.5 rounded-xl border border-border bg-raised p-1 shadow-floating">
      <span className="flex items-center gap-1 px-2 text-xs text-gray-9">
        <IconPlusOutline18 className="size-3.5" />
        Add
      </span>
      {items.map((item) => (
        <button
          key={item.label}
          type="button"
          onClick={item.run}
          className="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs text-gray-12 hover:bg-grayA-3 [&_svg]:size-3.5 [&_svg]:text-gray-11"
        >
          {item.icon}
          {item.label}
        </button>
      ))}
    </div>
  );
}

const AGENT_PROMPT =
  "Set up Unkey in my project. Fetch https://unkey.com/agent/setup.md and follow it.";

function AgentHint() {
  const [copied, setCopied] = useState(false);
  return (
    <div className="absolute bottom-20 left-1/2 flex -translate-x-1/2 items-center gap-3 text-xs text-gray-11">
      <IconSquareTerminalOutline18 className="size-3.5 text-gray-9" />
      <span>Or paste one prompt into Claude, Cursor or Codex.</span>
      <button
        type="button"
        onClick={() => {
          navigator.clipboard?.writeText(AGENT_PROMPT);
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1800);
        }}
        className="rounded-md border border-border bg-raised px-2 py-1 text-gray-12 hover:border-strong"
      >
        {copied ? "Copied" : "Copy prompt"}
      </button>
    </div>
  );
}
