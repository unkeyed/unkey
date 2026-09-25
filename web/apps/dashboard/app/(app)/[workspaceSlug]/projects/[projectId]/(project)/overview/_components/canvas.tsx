"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
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
import {
  type ReactNode,
  type RefObject,
  createContext,
  useContext,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { type OverviewModel, ago, compact } from "./overview-model";
import { useOverviewWindow } from "./overview-window";

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

type Hover = { kind: "app"; appId: string } | { kind: "keyspace"; keyAuthId: string } | null;

type Highlight = { apps: Set<string>; keyspaces: Set<string> };

const NONE: Highlight = { apps: new Set(), keyspaces: new Set() };

function highlightFor(hover: Hover, links: ProjectOverview["keyspaceLinks"]): Highlight {
  if (!hover) {
    return NONE;
  }
  if (hover.kind === "app") {
    return {
      apps: new Set([hover.appId]),
      keyspaces: new Set(links.filter((l) => l.appId === hover.appId).map((l) => l.keyAuthId)),
    };
  }
  return {
    apps: new Set(links.filter((l) => l.keyAuthId === hover.keyAuthId).map((l) => l.appId)),
    keyspaces: new Set([hover.keyAuthId]),
  };
}

const HoverContext = createContext<{ highlight: Highlight; setHover: (h: Hover) => void }>({
  highlight: NONE,
  setHover: () => undefined,
});

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
  const [hover, setHover] = useState<Hover>(null);
  const highlight = useMemo(
    () => highlightFor(hover, data.keyspaceLinks),
    [hover, data.keyspaceLinks],
  );
  const rootRef = useRef<HTMLDivElement>(null);
  const wires = useWires(rootRef, hover, data.keyspaceLinks);

  return (
    <HoverContext.Provider value={{ highlight, setHover }}>
      <div className="relative h-[520px] w-full overflow-hidden rounded-xl border border-border bg-gray-3/40">
        <div className="h-full overflow-x-auto overflow-y-hidden">
          <div
            ref={rootRef}
            className="relative flex h-full w-full min-w-[860px] items-start px-5 pt-5 pb-5"
          >
            <Wires wires={wires} />
            {hasApps ? (
              <>
                <Group
                  icon={<IconCubeOutline18 />}
                  label={`Apps · ${data.apps.length}`}
                  href={links.allApps}
                >
                  {[...data.apps]
                    .sort((a, b) => appRank(a) - appRank(b))
                    .map((app) => (
                      <AppCard
                        key={app.id}
                        app={app}
                        href={links.app(app.id)}
                        active={highlight.apps.has(app.id)}
                      />
                    ))}
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

            <Connector dashed={data.keyspaceLinks.length === 0} />

            <Group
              icon={<IconGridOutline18 />}
              label={
                hasKeyspaces || hasRatelimits
                  ? `Services · ${data.keyspaces.length + data.ratelimits.length}`
                  : "Services"
              }
            >
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
    </HoverContext.Provider>
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
  const scrollRef = useRef<HTMLDivElement>(null);
  const more = useMoreBelow(scrollRef);
  return (
    <div className="flex max-h-full min-w-0 max-w-[340px] flex-1 flex-col gap-2 rounded-xl border border-grayA-3 bg-grayA-2 p-2.5">
      <div className="flex shrink-0 items-center justify-between pr-[7px] pb-0.5 pl-1">
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
      <div
        ref={scrollRef}
        data-wire-scroll
        className={cn(
          "-my-1 -mr-2.5 -ml-1 flex min-h-0 border-r-4 border-transparent flex-col gap-2 overflow-y-auto py-1 pr-[2px] pl-1 [scrollbar-gutter:stable] [scrollbar-color:transparent_transparent] [scrollbar-width:thin] hover:[scrollbar-color:var(--color-grayA-6)_transparent]",
          more && "[mask-image:linear-gradient(to_bottom,black_calc(100%-56px),transparent)]",
        )}
      >
        {children}
      </div>
    </div>
  );
}

function useMoreBelow(ref: RefObject<HTMLDivElement | null>) {
  const [more, setMore] = useState(false);
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) {
      return;
    }
    const check = () => setMore(el.scrollTop + el.clientHeight < el.scrollHeight - 4);
    check();
    const observer = new ResizeObserver(check);
    observer.observe(el);
    el.addEventListener("scroll", check);
    return () => {
      observer.disconnect();
      el.removeEventListener("scroll", check);
    };
  }, [ref]);
  return more;
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
  default:
    "border-border bg-raised [--divider:var(--hairline)] hover:border-gray-10 hover:shadow-[0_0_0_3px_var(--color-grayA-3)] data-[active=true]:border-gray-10 data-[active=true]:shadow-[0_0_0_3px_var(--color-grayA-3)]",
  error:
    "border-error-6 bg-error-2 [--divider:var(--color-error-6)] hover:border-error-9 hover:shadow-[0_0_0_3px_var(--color-errorA-3)] data-[active=true]:border-error-9 data-[active=true]:shadow-[0_0_0_3px_var(--color-errorA-3)]",
  warning:
    "border-warning-6 bg-warning-2 [--divider:var(--color-warning-6)] hover:border-warning-9 hover:shadow-[0_0_0_3px_var(--color-warningA-3)] data-[active=true]:border-warning-9 data-[active=true]:shadow-[0_0_0_3px_var(--color-warningA-3)]",
};

function Card({
  children,
  href,
  tone = "default",
  active = false,
  ...rest
}: {
  children: ReactNode;
  href?: Route;
  tone?: Tone;
  active?: boolean;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
  "data-wire-app"?: string;
}) {
  const cls = cn(
    "block rounded-lg border shadow-xs transition-[border-color,box-shadow]",
    TONE[tone],
  );
  return href ? (
    <Link href={href} className={cls} data-active={active} {...rest}>
      {children}
    </Link>
  ) : (
    <div className={cls} data-active={active} {...rest}>
      {children}
    </div>
  );
}

function CardHeader({ icon, title, right }: { icon: ReactNode; title: string; right?: ReactNode }) {
  return (
    <div className="flex items-center gap-2 px-3 py-2 leading-5 not-last:border-b [border-color:var(--divider)]">
      <span className="text-gray-11 [&_svg]:size-4">{icon}</span>
      <span className="min-w-0 truncate text-[13px] font-medium text-gray-12">{title}</span>
      {right && <span className="ml-auto shrink-0">{right}</span>}
    </div>
  );
}

function AppCard({ app, href, active }: { app: OverviewApp; href: Route; active: boolean }) {
  const { setHover } = useContext(HoverContext);
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
    <Card
      href={href}
      tone={tone}
      active={active}
      data-wire-app={app.id}
      onMouseEnter={() => setHover({ kind: "app", appId: app.id })}
      onMouseLeave={() => setHover(null)}
    >
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

const METRIC_COLS = "grid grid-cols-[minmax(0,1fr)_52px_72px] items-center gap-x-3";

function MetricHeader({
  icon,
  title,
  href,
  columns,
}: {
  icon: ReactNode;
  title: string;
  href: Route;
  columns: [string, string];
}) {
  return (
    <div className={cn(METRIC_COLS, "px-3 py-2.5 not-last:border-b [border-color:var(--divider)]")}>
      <Link href={href} className="flex min-w-0 items-center gap-2 hover:text-gray-12">
        <span className="text-gray-11 [&_svg]:size-4">{icon}</span>
        <span className="truncate text-[13px] font-medium text-gray-12">{title}</span>
      </Link>
      <span className="text-right text-[11px] text-gray-9">{columns[0]}</span>
      <span className="text-right text-[11px] text-gray-9">{columns[1]}</span>
    </div>
  );
}

function MetricRow({
  href,
  name,
  values,
  active = false,
  wireId,
  onMouseEnter,
  onMouseLeave,
}: {
  wireId?: string;
  href: Route;
  name: string;
  values: [string, string];
  active?: boolean;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
}) {
  return (
    <Link
      href={href}
      data-wire-ks={wireId}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      className={cn(
        METRIC_COLS,
        "px-3 py-1.5 text-xs hover:bg-grayA-2",
        active && "bg-grayA-3 hover:bg-grayA-3",
      )}
    >
      <span className="truncate font-medium text-gray-12">{name}</span>
      <span className="text-right text-gray-11 tabular-nums">{values[0]}</span>
      <span className="text-right text-gray-11 tabular-nums">{values[1]}</span>
    </Link>
  );
}

function KeyspacesCard({ data, links }: { data: ProjectOverview; links: CanvasLinks }) {
  const { highlight } = useContext(HoverContext);
  return (
    <Card active={data.keyspaces.some((k) => highlight.keyspaces.has(k.keyAuthId))}>
      <MetricHeader
        icon={<IconNodesOutline18 />}
        title="Keyspaces"
        href={links.allKeyspaces}
        columns={["Keys", "Verified 7d"]}
      />
      <div className="py-1">
        {data.keyspaces.map((ks) => (
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
  const window = useOverviewWindow();
  const { data } = trpc.api.overview.timeseries.useQuery(
    { keyspaceId: keyAuthId, ...window, since: "" },
    { trpc: { context: { skipBatch: true } } },
  );
  const total = data?.timeseries?.reduce((a, p) => a + p.y.total, 0);
  const { highlight, setHover } = useContext(HoverContext);
  return (
    <MetricRow
      active={highlight.keyspaces.has(keyAuthId)}
      wireId={keyAuthId}
      onMouseEnter={() => setHover({ kind: "keyspace", keyAuthId })}
      onMouseLeave={() => setHover(null)}
      href={href}
      name={name}
      values={[compact(keyCount), total == null ? "…" : compact(total)]}
    />
  );
}

function RatelimitsCard({ data, links }: { data: ProjectOverview; links: CanvasLinks }) {
  const window = useOverviewWindow();
  const shown = data.ratelimits;
  const { data: ts } = trpc.ratelimit.logs.queryRatelimitTimeseriesBatch.useQuery({
    namespaceIds: shown.map((n) => n.id),
    ...window,
  });
  return (
    <Card>
      <MetricHeader
        icon={<IconGaugeOutline18 />}
        title="Ratelimits"
        href={links.allRatelimits}
        columns={["Requests", "Blocked"]}
      />
      <div className="py-1">
        {shown.map((ns) => {
          const series = ts?.timeseriesByNamespace[ns.id] ?? [];
          const total = series.reduce((a, p) => a + p.y.total, 0);
          const passed = series.reduce((a, p) => a + p.y.passed, 0);
          const blocked = total ? ((total - passed) / total) * 100 : 0;
          return (
            <MetricRow
              key={ns.id}
              href={links.ratelimit(ns.id)}
              name={ns.name}
              values={ts ? [compact(total), `${blocked.toFixed(1)}%`] : ["…", "…"]}
            />
          );
        })}
      </div>
    </Card>
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
      className="group flex items-start gap-2.5 rounded-lg border border-border bg-raised px-3 py-3 text-left shadow-xs transition-[border-color,box-shadow] hover:border-gray-10 hover:shadow-[0_0_0_3px_var(--color-grayA-3)]"
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
        <DropdownMenuGroup>
          <DropdownMenuLabel>{label}</DropdownMenuLabel>
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
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

type Wire = { id: string; d: string };

const HEADER_MID_PX = 18;

function useWires(
  rootRef: RefObject<HTMLDivElement | null>,
  hover: Hover,
  links: ProjectOverview["keyspaceLinks"],
) {
  const [wires, setWires] = useState<Wire[]>([]);

  useLayoutEffect(() => {
    const root = rootRef.current;
    const pairs = hover
      ? hover.kind === "app"
        ? links.filter((l) => l.appId === hover.appId)
        : links.filter((l) => l.keyAuthId === hover.keyAuthId)
      : [];
    if (!root || pairs.length === 0) {
      setWires([]);
      return;
    }
    const inView = (el: HTMLElement, y: number) => {
      const viewport = el.closest("[data-wire-scroll]")?.getBoundingClientRect();
      return viewport ? y >= viewport.top && y <= viewport.bottom : false;
    };
    const measure = () => {
      const base = root.getBoundingClientRect();
      const next: Wire[] = [];
      for (const { appId, keyAuthId } of pairs) {
        const app = root.querySelector<HTMLElement>(`[data-wire-app="${appId}"]`);
        const row = root.querySelector<HTMLElement>(`[data-wire-ks="${keyAuthId}"]`);
        if (!app || !row) {
          continue;
        }
        const a = app.getBoundingClientRect();
        const r = row.getBoundingClientRect();
        const y1 = a.top + HEADER_MID_PX;
        const y2 = r.top + r.height / 2;
        if (!inView(app, y1) || !inView(row, y2)) {
          continue;
        }
        const x1 = a.right - base.left;
        const x2 = r.left - base.left;
        const mid = Math.round((x1 + x2) / 2);
        next.push({
          id: `${appId}:${keyAuthId}`,
          d: `M ${x1} ${y1 - base.top} H ${mid} V ${y2 - base.top} H ${x2}`,
        });
      }
      setWires(next);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(root);
    root.addEventListener("scroll", measure, true);
    return () => {
      observer.disconnect();
      root.removeEventListener("scroll", measure, true);
    };
  }, [rootRef, hover, links]);

  return wires;
}

function Wires({ wires }: { wires: Wire[] }) {
  if (wires.length === 0) {
    return null;
  }
  return (
    <svg
      className="pointer-events-none absolute inset-0 z-10 h-full w-full overflow-visible"
      aria-hidden
    >
      <g
        fill="none"
        strokeLinejoin="round"
        className="stroke-gray-11"
        strokeWidth={1}
        shapeRendering="crispEdges"
      >
        {wires.map((w) => (
          <path key={w.id} d={w.d} />
        ))}
      </g>
    </svg>
  );
}
