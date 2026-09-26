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
  useRef,
  useState,
} from "react";
import { type OverviewModel, ago, compact } from "../../overview/_components/overview-model";
import { WINDOW, useKeyspaceTotal } from "./totals";

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

export function appRank(app: OverviewApp): number {
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

type Ctx = {
  highlight: Highlight;
  setHover: (h: Hover) => void;
  matches: (text: string) => boolean;
  onSelectApp?: (id: string) => void;
};

const HoverContext = createContext<Ctx>({
  highlight: NONE,
  setHover: () => undefined,
  matches: () => true,
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

export type CanvasOptions = {
  order?: "apps-first" | "services-first";
  arrangement?: "columns" | "lanes";
  frame?: "card" | "bleed";
  heightClass?: string;
  inbound?: ReactNode;
  overlay?: ReactNode;
  dockExtra?: ReactNode;
  dock?: boolean;
  query?: string;
  selectedAppId?: string | null;
  onSelectApp?: (id: string) => void;
};

export function ProtoCanvas({
  data,
  model,
  links,
  actions,
  options = {},
}: {
  data: ProjectOverview;
  model: OverviewModel;
  links: CanvasLinks;
  actions: CanvasActions;
  options?: CanvasOptions;
}) {
  const {
    order = "apps-first",
    arrangement = "columns",
    frame = "card",
    heightClass = "h-[520px]",
    dock = true,
  } = options;
  const [hover, setHover] = useState<Hover>(null);
  const effective: Hover =
    hover ?? (options.selectedAppId ? { kind: "app", appId: options.selectedAppId } : null);
  const highlight = highlightFor(effective, data.keyspaceLinks);
  const rootRef = useRef<HTMLDivElement>(null);
  const wires = useWires(rootRef, effective, data.keyspaceLinks, arrangement);
  const q = (options.query ?? "").trim().toLowerCase();
  const matches = (text: string) => q === "" || text.toLowerCase().includes(q);

  const apps = (
    <AppsGroup
      data={data}
      model={model}
      links={links}
      actions={actions}
      grid={arrangement === "lanes"}
    />
  );
  const services = (
    <ServicesGroup data={data} links={links} actions={actions} grid={arrangement === "lanes"} />
  );
  const linked = data.keyspaceLinks.length > 0;

  return (
    <HoverContext.Provider
      value={{ highlight, setHover, matches, onSelectApp: options.onSelectApp }}
    >
      <div
        className={cn(
          "relative w-full overflow-hidden bg-gray-2",
          heightClass,
          frame === "card" && "rounded-xl border border-border",
        )}
        style={{
          backgroundImage: "radial-gradient(var(--color-grayA-5) 1px, transparent 1px)",
          backgroundSize: "14px 14px",
        }}
      >
        <div className="h-full overflow-x-auto overflow-y-hidden">
          {arrangement === "lanes" ? (
            <div
              ref={rootRef}
              className="relative flex h-full w-full min-w-[860px] flex-col items-start gap-0 p-5"
            >
              <Wires wires={wires} />
              <div data-wire-lane className="flex max-h-[48%] w-full min-h-0 shrink-0">
                {services}
              </div>
              <Connector vertical dashed={!linked} />
              <div className="flex min-h-0 w-full flex-1 pb-16">{apps}</div>
            </div>
          ) : (
            <div
              ref={rootRef}
              className={cn(
                "relative flex h-full w-full min-w-[860px] items-start p-5",
                options.overlay && "pt-16",
              )}
            >
              <Wires wires={wires} />
              {options.inbound && (
                <>
                  {options.inbound}
                  <Connector />
                </>
              )}
              {order === "apps-first" ? apps : services}
              <Connector dashed={!linked} />
              {order === "apps-first" ? services : apps}
            </div>
          )}
        </div>
        {options.overlay && <div className="absolute top-4 left-5 z-20">{options.overlay}</div>}
        {dock && <CommandBar actions={actions} extra={options.dockExtra} />}
      </div>
    </HoverContext.Provider>
  );
}

function AppsGroup({
  data,
  model,
  links,
  actions,
  grid,
}: {
  data: ProjectOverview;
  model: OverviewModel;
  links: CanvasLinks;
  actions: CanvasActions;
  grid: boolean;
}) {
  const { highlight } = useContext(HoverContext);
  if (data.apps.length === 0) {
    return (
      <Group icon={<IconCubeOutline18 />} label="Apps" grid={grid}>
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
    );
  }
  return (
    <Group
      icon={<IconCubeOutline18 />}
      label={`Apps · ${data.apps.length}`}
      href={links.allApps}
      grid={grid}
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
  );
}

function ServicesGroup({
  data,
  links,
  actions,
  grid,
}: {
  data: ProjectOverview;
  links: CanvasLinks;
  actions: CanvasActions;
  grid: boolean;
}) {
  const hasKeyspaces = data.keyspaces.length > 0;
  const hasRatelimits = data.ratelimits.length > 0;
  return (
    <Group
      icon={<IconGridOutline18 />}
      label={
        hasKeyspaces || hasRatelimits
          ? `Services · ${data.keyspaces.length + data.ratelimits.length}`
          : "Services"
      }
      grid={grid}
      columns={2}
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
  );
}

export function Group({
  icon,
  label,
  href,
  grid = false,
  columns = 3,
  className,
  children,
}: {
  icon: ReactNode;
  label: string;
  href?: Route;
  grid?: boolean;
  columns?: 2 | 3;
  className?: string;
  children: ReactNode;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const more = useMoreBelow(scrollRef);
  return (
    <div
      className={cn(
        "flex max-h-full min-w-0 flex-col gap-2 rounded-xl bg-grayA-2 p-2.5 backdrop-blur-sm",
        grid ? "w-full" : "max-w-[340px] flex-1",
        className,
      )}
    >
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
          "-my-1 -mr-2.5 -ml-1 min-h-0 border-r-4 border-transparent overflow-y-auto py-1 pr-[2px] pl-1 [scrollbar-gutter:stable] [scrollbar-color:transparent_transparent] [scrollbar-width:thin] hover:[scrollbar-color:var(--color-grayA-6)_transparent]",
          grid
            ? cn("grid items-start gap-2", columns === 2 ? "grid-cols-2" : "grid-cols-3")
            : "flex flex-col gap-2",
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

export function Connector({
  dashed = false,
  vertical = false,
}: { dashed?: boolean; vertical?: boolean }) {
  return (
    <div
      className={cn(
        "shrink-0",
        vertical ? "ml-[178px] h-6 border-l" : "mt-[58px] w-8 border-t",
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
  onClick,
  tone = "default",
  active = false,
  dim = false,
  ...rest
}: {
  children: ReactNode;
  href?: Route;
  onClick?: () => void;
  tone?: Tone;
  active?: boolean;
  dim?: boolean;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
  "data-wire-app"?: string;
  "data-wire-kscard"?: boolean;
}) {
  const cls = cn(
    "block w-full rounded-lg border text-left shadow-xs transition-[border-color,box-shadow,opacity]",
    TONE[tone],
    dim && "opacity-35",
  );
  if (onClick) {
    return (
      <button type="button" onClick={onClick} className={cls} data-active={active} {...rest}>
        {children}
      </button>
    );
  }
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

export function sourceIcon(app: OverviewApp) {
  return app.sourceType === "oci" ? (
    <IconLayers2Outline18 />
  ) : app.repositoryFullName ? (
    <Github />
  ) : (
    <IconTerminalOutline18 />
  );
}

export function StatusLabel({ app }: { app: OverviewApp }) {
  const d = app.latest;
  if (!d) {
    return <span className="text-xs text-gray-9">Not deployed</span>;
  }
  const tone = toneOf(app);
  return (
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
  );
}

function toneOf(app: OverviewApp): Tone {
  const s = app.latest?.status;
  return s === "failed" ? "error" : s === "awaiting_approval" ? "warning" : "default";
}

function AppCard({ app, href, active }: { app: OverviewApp; href: Route; active: boolean }) {
  const { setHover, matches, onSelectApp } = useContext(HoverContext);
  const d = app.latest;
  return (
    <Card
      href={href}
      onClick={onSelectApp ? () => onSelectApp(app.id) : undefined}
      tone={toneOf(app)}
      active={active}
      dim={!matches(`${app.name} ${app.repositoryFullName ?? ""} ${d?.branch ?? ""}`)}
      data-wire-app={app.id}
      onMouseEnter={() => setHover({ kind: "app", appId: app.id })}
      onMouseLeave={() => setHover(null)}
    >
      <CardHeader icon={sourceIcon(app)} title={app.name} right={<StatusLabel app={app} />} />
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
  const { matches } = useContext(HoverContext);
  return (
    <Link
      href={href}
      data-wire-ks={wireId}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      className={cn(
        METRIC_COLS,
        "px-3 py-1.5 text-xs transition-opacity hover:bg-grayA-2",
        active && "bg-grayA-3 hover:bg-grayA-3",
        !matches(name) && "opacity-35",
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
    <Card
      data-wire-kscard
      active={data.keyspaces.some((k) => highlight.keyspaces.has(k.keyAuthId))}
    >
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
  const total = useKeyspaceTotal(keyAuthId);
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
  const shown = data.ratelimits;
  const { data: ts } = trpc.ratelimit.logs.queryRatelimitTimeseriesBatch.useQuery({
    namespaceIds: shown.map((n) => n.id),
    ...WINDOW,
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

export type MenuEntry = { label: string; description: string; icon: ReactNode; run: () => void };

export function appEntries(actions: CanvasActions): MenuEntry[] {
  return [
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
}

export function serviceEntries(actions: CanvasActions): MenuEntry[] {
  return [
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
}

function CommandBar({ actions, extra }: { actions: CanvasActions; extra?: ReactNode }) {
  return (
    <div className="absolute bottom-4 left-1/2 z-20 flex -translate-x-1/2 items-center gap-0.5 rounded-xl bg-raised p-1 shadow-floating">
      <AddMenu label="Add app" icon={<IconCubeOutline18 />} entries={appEntries(actions)} />
      <AddMenu label="Add service" icon={<IconGridOutline18 />} entries={serviceEntries(actions)} />
      {extra && (
        <>
          <span className="mx-0.5 h-4 w-px bg-grayA-4" />
          {extra}
        </>
      )}
    </div>
  );
}

export function AddMenu({
  label,
  icon,
  entries,
  side = "top",
}: {
  label: string;
  icon: ReactNode;
  entries: MenuEntry[];
  side?: "top" | "bottom";
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
      <DropdownMenuContent side={side} align="center" sideOffset={8} className="w-72 p-1">
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
  arrangement: "columns" | "lanes",
) {
  const [wires, setWires] = useState<Wire[]>([]);
  const hoverAppId = hover?.kind === "app" ? hover.appId : null;
  const hoverKeyAuthId = hover?.kind === "keyspace" ? hover.keyAuthId : null;

  useLayoutEffect(() => {
    const root = rootRef.current;
    const pairs = hoverAppId
      ? links.filter((l) => l.appId === hoverAppId)
      : hoverKeyAuthId
        ? links.filter((l) => l.keyAuthId === hoverKeyAuthId)
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
      const lane = root.querySelector<HTMLElement>("[data-wire-lane]")?.getBoundingClientRect();
      const ksCard = root.querySelector<HTMLElement>("[data-wire-kscard]")?.getBoundingClientRect();
      const next: Wire[] = [];
      for (const { appId, keyAuthId } of pairs) {
        const app = root.querySelector<HTMLElement>(`[data-wire-app="${appId}"]`);
        const row = root.querySelector<HTMLElement>(`[data-wire-ks="${keyAuthId}"]`);
        if (!app || !row) {
          continue;
        }
        const a = app.getBoundingClientRect();
        const r = row.getBoundingClientRect();
        const y2 = r.top + r.height / 2 - base.top;
        if (!inView(row, r.top + r.height / 2)) {
          continue;
        }
        if (arrangement === "lanes" && lane && ksCard) {
          if (!inView(app, a.top + 4)) {
            continue;
          }
          const x1 = Math.round(a.left + a.width / 2 - base.left);
          const y1 = a.top - base.top;
          const band = lane.bottom - base.top + 12;
          const gutter = ksCard.left - base.left - 6;
          next.push({
            id: `${appId}:${keyAuthId}`,
            d: `M ${x1} ${y1} V ${band} H ${gutter} V ${y2} H ${r.left - base.left}`,
          });
          continue;
        }
        const y1 = a.top + HEADER_MID_PX;
        if (!inView(app, y1)) {
          continue;
        }
        const appRight = a.left < r.left;
        const x1 = appRight ? a.right - base.left : a.left - base.left;
        const x2 = appRight ? r.left - base.left : r.right - base.left;
        const mid = Math.round((x1 + x2) / 2);
        next.push({
          id: `${appId}:${keyAuthId}`,
          d: `M ${x1} ${y1 - base.top} H ${mid} V ${y2} H ${x2}`,
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
  }, [rootRef, hoverAppId, hoverKeyAuthId, links, arrangement]);

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
