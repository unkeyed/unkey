"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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
  IconChevronDownOutline18,
  IconCodeBranchOutline18,
  IconCubeOutline18,
  IconFingerprintOutline18,
  IconGaugeOutline18,
  IconGridOutline18,
  IconLayers2Outline18,
  IconNodesOutline18,
  IconPlusOutline18,
  IconShieldKeyOutline18,
  IconTerminalOutline18,
} from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import { useMemo } from "react";
import { type OverviewModel, ago, compact } from "./overview-model";

const WINDOW_MS = 7 * 24 * 60 * 60 * 1000;
const MAX_APPS = 4;

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
};

export type CanvasActions = {
  createApp: () => void;
  createKeyspace: () => void;
  createRatelimit: () => void;
  openIdentities: () => void;
  openPermissions: () => void;
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
      className="relative h-[520px] w-full overflow-hidden rounded-xl border border-border bg-gray-2"
      style={{
        backgroundImage: "radial-gradient(var(--color-grayA-5) 1px, transparent 1px)",
        backgroundSize: "14px 14px",
      }}
    >
      <div className="h-full overflow-auto">
        <div className="flex w-full min-w-[860px] items-start px-5 pt-5 pb-16">
          {hasApps ? (
            <>
              <Group
                icon={<IconCubeOutline18 />}
                label={`Apps · ${data.apps.length}`}
                href={links.allApps}
              >
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
            <Group icon={<IconCubeOutline18 />} label="Apps">
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

          <Group icon={<IconGridOutline18 />} label="Services">
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
      </div>
      <CommandBar actions={actions} />
    </div>
  );
}

function Group({
  icon,
  label,
  href,
  children,
}: {
  icon: ReactNode;
  label: string;
  href?: Route;
  children: ReactNode;
}) {
  return (
    <div className="flex min-w-0 max-w-[340px] flex-1 flex-col gap-2 rounded-xl bg-grayA-2 p-2.5 backdrop-blur-sm">
      <div className="flex items-center justify-between px-1 pb-0.5">
        <span className="flex items-center gap-1.5 text-xs text-gray-11 [&_svg]:size-3.5">
          {icon}
          {label}
        </span>
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

type Tone = "default" | "error" | "warning";

const TONE: Record<Tone, string> = {
  default: "border-border bg-raised hover:border-strong",
  error: "border-error-6 bg-error-2 hover:border-error-8",
  warning: "border-warning-6 bg-warning-2 hover:border-warning-8",
};

function Card({
  children,
  href,
  tone = "default",
}: {
  children: ReactNode;
  href?: Route;
  tone?: Tone;
}) {
  const cls = cn("block rounded-lg border shadow-xs transition-colors", TONE[tone]);
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
    <div className="flex items-center gap-2 px-3 py-2.5 not-last:border-b [border-color:inherit]">
      <span className="text-gray-11 [&_svg]:size-4">{icon}</span>
      <span className="min-w-0 truncate text-[13px] font-medium text-gray-12">{title}</span>
      {right && <span className="ml-auto shrink-0">{right}</span>}
    </div>
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
  const tone: Tone =
    d?.status === "failed" ? "error" : d?.status === "awaiting_approval" ? "warning" : "default";
  return (
    <Card href={href} tone={tone}>
      <CardHeader
        icon={icon}
        title={app.name}
        right={
          d ? (
            <span
              className={cn(
                "inline-flex items-center gap-1.5 text-xs",
                tone === "error"
                  ? "text-error-11"
                  : tone === "warning"
                    ? "text-warning-11"
                    : "text-gray-11",
              )}
            >
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
      ) : null}
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
  const total = data?.timeseries?.reduce((a, p) => a + p.y.total, 0);
  return (
    <Link href={href} className="block px-3 py-1.5 hover:bg-grayA-2">
      <div className="truncate text-xs text-gray-12">{name}</div>
      <Stats
        items={[
          { label: "Keys", value: compact(keyCount) },
          { label: "Verifications", value: total == null ? "…" : compact(total) },
        ]}
      />
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
              className="block px-3 py-1.5 hover:bg-grayA-2"
            >
              <div className="truncate text-xs text-gray-12">{ns.name}</div>
              <Stats
                items={[
                  { label: "Requests", value: ts ? compact(total) : "…" },
                  {
                    label: "Blocked",
                    value: ts ? `${blocked.toFixed(1)}%` : "…",
                    warn: blocked > 5,
                  },
                ]}
              />
            </Link>
          );
        })}
      </div>
    </Card>
  );
}

function Stats({
  items,
}: {
  items: Array<{ label: string; value: string; warn?: boolean }>;
}) {
  return (
    <div className="flex items-center gap-1.5 text-[11px]">
      {items.map((item, i) => (
        <span key={item.label} className="flex items-center gap-1.5">
          {i > 0 && <span className="text-gray-7">·</span>}
          <span className="text-gray-11">{item.label}</span>
          <span className={cn(item.warn ? "text-warning-11" : "text-gray-9")}>{item.value}</span>
        </span>
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
      className="group flex items-start gap-2.5 rounded-lg border border-border bg-raised px-3 py-3 text-left transition-colors hover:border-strong"
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

type MenuEntry = { label: string; description: string; icon: ReactNode; run: () => void };

function CommandBar({ actions }: { actions: CanvasActions }) {
  const apps: MenuEntry[] = [
    {
      label: "GitHub repository",
      description: "Deploy on every push",
      icon: <Github />,
      run: actions.createApp,
    },
    {
      label: "Container image",
      description: "Deploy from any OCI registry",
      icon: <IconLayers2Outline18 />,
      run: actions.createApp,
    },
  ];
  const services: MenuEntry[] = [
    {
      label: "Keyspace",
      description: "Issue and verify API keys",
      icon: <IconNodesOutline18 />,
      run: actions.createKeyspace,
    },
    {
      label: "Ratelimit",
      description: "Cap requests per user, key or IP",
      icon: <IconGaugeOutline18 />,
      run: actions.createRatelimit,
    },
    {
      label: "Identities",
      description: "Group keys and limits by user",
      icon: <IconFingerprintOutline18 />,
      run: actions.openIdentities,
    },
    {
      label: "Permissions",
      description: "Roles and permissions on keys",
      icon: <IconShieldKeyOutline18 />,
      run: actions.openPermissions,
    },
  ];
  return (
    <div className="absolute bottom-4 left-1/2 flex -translate-x-1/2 items-center gap-0.5 rounded-xl bg-raised p-1 shadow-floating">
      <AddMenu label="Add app" icon={<IconCubeOutline18 />} entries={apps} />
      <AddMenu label="Add service" icon={<IconGridOutline18 />} entries={services} />
    </div>
  );
}

function AddMenu({
  label,
  icon,
  entries,
}: {
  label: string;
  icon: ReactNode;
  entries: MenuEntry[];
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={label}
        className="flex h-8 items-center gap-1 rounded-lg px-2 text-gray-11 hover:bg-grayA-3 hover:text-gray-12 data-popup-open:bg-grayA-3 data-popup-open:text-gray-12 [&_svg]:size-4"
      >
        {icon}
        <IconChevronDownOutline18 className="size-3! text-gray-9" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="center" sideOffset={8} className="w-72 p-1">
        {entries.map((e) => (
          <DropdownMenuItem
            key={e.label}
            onClick={e.run}
            className="cursor-pointer gap-3 px-2 py-2"
          >
            <span className="flex size-8 shrink-0 items-center justify-center rounded-md border border-border bg-background text-gray-11 [&_svg]:size-4">
              {e.icon}
            </span>
            <span className="min-w-0">
              <span className="block text-[13px] font-medium text-gray-12">{e.label}</span>
              <span className="block text-xs text-gray-9">{e.description}</span>
            </span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
