"use client";

import { NavItems } from "@/components/navigation/sidebar/app-sidebar/components/nav-items";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarMenu,
  useSidebar,
} from "@/components/ui/sidebar";
import { useNavigationContext } from "@/hooks/use-navigation-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Quotas, Workspace } from "@/lib/db";
import { KEYSPACE_LABEL } from "@/lib/keyspace-label";
import { cn } from "@/lib/utils";
import {
  ArrowOppositeDirectionY,
  BracketsSquareDots,
  ChevronRight,
  Coins,
  Cube,
  Fingerprint,
  Gauge,
  Gear,
  Grid,
  InputSearch,
  Layers3,
  Nodes,
  ShieldKey,
  SidebarLeftHide,
  SidebarLeftShow,
} from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { usePathname, useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";
import { LeafSidebarView } from "./shared/leaf-sidebar-view";
import { V2BTopHeader, V2B_HEADER_HEIGHT } from "./shared/v2b-header";

type Props = React.ComponentProps<typeof Sidebar> & {
  workspace: Workspace & { quotas: Quotas | null };
};

/**
 * v2b — full-width breadcrumb top header + contextual left sidebar.
 *
 * Mirror of v2's leaf logic (ProjectLeaf / AppLeaf / SettingsLeaf /
 * AuthorizationLeaf / UnifiedTree) minus the in-sidebar back links and
 * the sidebar-level workspace switcher (both moved to the top header).
 */
export function V2bVariant(props: Props) {
  const context = useNavigationContext();
  const rawSegments = useSelectedLayoutSegments();
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);
  const pathname = usePathname() ?? "";
  const isOnSettings = segments.at(1) === "settings";
  const isOnAuthorization = segments.at(1) === "authorization";

  const content = (() => {
    if (context.type === "resource" && context.resourceType === "project") {
      const appMatch = pathname.match(/\/projects\/[^/]+\/apps\/([^/]+)/);
      if (appMatch?.[1]) {
        return <AppLeaf projectId={context.resourceId} appSlug={appMatch[1]} />;
      }
      return <ProjectLeaf projectId={context.resourceId} />;
    }
    if (context.type === "resource") {
      return <LeafSidebarView context={context} />;
    }
    if (isOnSettings) {
      return <SettingsLeaf />;
    }
    if (isOnAuthorization) {
      return <AuthorizationLeaf />;
    }
    return <UnifiedTree />;
  })();

  return (
    <>
      <V2BTopHeader />
      <Sidebar
        {...props}
        collapsible="icon"
        className={cn(
          "[&_[data-sidebar=sidebar]]:bg-background",
          props.className,
        )}
        style={
          {
            top: V2B_HEADER_HEIGHT,
            height: `calc(100svh - ${V2B_HEADER_HEIGHT}px)`,
            // Contextual rail; the icon-collapsed rail is intentionally
            // tight so it reads as a skinny icon strip rather than a
            // half-hidden sidebar.
            "--sidebar-width": "13rem",
            "--sidebar-width-icon": "3rem",
          } as React.CSSProperties
        }
      >
        <SidebarContent>{content}</SidebarContent>
        <SidebarFooter className="mx-0 gap-0 border-t-0 p-2">
          <CollapseButton />
        </SidebarFooter>
      </Sidebar>
    </>
  );
}

function CollapseButton() {
  const { state, toggleSidebar } = useSidebar();
  const collapsed = state === "collapsed";
  const Icon = collapsed ? SidebarLeftShow : SidebarLeftHide;
  const label = collapsed ? "Expand sidebar" : "Collapse sidebar";
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={toggleSidebar}
          aria-label={label}
          className="flex size-8 items-center justify-center rounded-md text-gray-11 hover:bg-gray-3 hover:text-gray-12"
        >
          <Icon iconSize="md-regular" className="shrink-0" />
        </button>
      </TooltipTrigger>
      <TooltipContent
        side="right"
        align="center"
        className="dark:bg-white bg-black text-gray-1 px-2 py-1 border border-accent-6 shadow-md font-medium text-xs"
      >
        {label}
      </TooltipContent>
    </Tooltip>
  );
}

function UnifiedTree() {
  const workspace = useWorkspaceNavigation();
  const rawSegments = useSelectedLayoutSegments();
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);
  const base = `/${workspace.slug}`;

  const drillTag = <ChevronRight className="size-2 text-gray-9" iconSize="sm-thin" />;

  const deployItem: NavItem = {
    icon: Cube,
    href: `${base}/projects`,
    label: "Projects",
    active: segments.at(1) === "projects" && !segments.at(2),
  };

  const featureItems: NavItem[] = [
    {
      icon: Nodes,
      href: `${base}/apis`,
      label: KEYSPACE_LABEL,
      active: segments.at(1) === "apis" && !segments.at(2),
    },
    {
      icon: Gauge,
      href: `${base}/ratelimits`,
      label: "Ratelimit",
      active: segments.at(1) === "ratelimits" && !segments.at(2),
    },
    {
      icon: ShieldKey,
      href: `${base}/authorization/roles`,
      label: "Authorization",
      active: segments.includes("authorization"),
      tag: drillTag,
    },
    {
      icon: Fingerprint,
      href: `${base}/identities`,
      label: "Identities",
      active: segments.at(1) === "identities" && !segments.at(2),
    },
    {
      icon: Layers3,
      href: `${base}/logs`,
      label: "Logs",
      active: segments.at(1) === "logs",
    },
  ];

  const workspaceItems: NavItem[] = [
    {
      icon: InputSearch,
      href: `${base}/audit`,
      label: "Audit Log",
      active: segments.at(1) === "audit",
    },
    {
      icon: Coins,
      href: `${base}/portal`,
      label: "Customer Billing",
      active: segments.at(1) === "portal",
      hidden: !workspace.betaFeatures?.portal,
    },
    {
      icon: Gear,
      href: `${base}/settings/general`,
      label: "Workspace Settings",
      active: segments.at(1) === "settings",
      tag: drillTag,
    },
  ].filter((item) => !item.hidden);

  return (
    <>
      <Section className="py-1.5">
        <SidebarMenu className="gap-1">
          <NavItems item={deployItem} />
        </SidebarMenu>
      </Section>

      <Section>
        <SidebarMenu className="gap-1">
          {featureItems.map((item) => (
            <NavItems key={item.href} item={item} />
          ))}
        </SidebarMenu>
      </Section>

      <Section last>
        <SidebarMenu className="gap-1">
          {workspaceItems.map((item) => (
            <NavItems key={item.href} item={item} />
          ))}
        </SidebarMenu>
      </Section>
    </>
  );
}

function AuthorizationLeaf() {
  const workspace = useWorkspaceNavigation();
  const rawSegments = useSelectedLayoutSegments();
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);
  const base = `/${workspace.slug}`;

  const items: NavItem[] = [
    {
      icon: null,
      href: `${base}/authorization/roles`,
      label: "Roles",
      active: segments.includes("roles"),
    },
    {
      icon: null,
      href: `${base}/authorization/permissions`,
      label: "Permissions",
      active: segments.includes("permissions"),
    },
  ];

  return (
    <div className="flex flex-col gap-2 px-2 py-2 group-data-[collapsible=icon]:px-0">
      <SidebarMenu className="gap-1">
        {items.map((item) => (
          <NavItems key={item.href} item={item} />
        ))}
      </SidebarMenu>
    </div>
  );
}

function SettingsLeaf() {
  const workspace = useWorkspaceNavigation();
  const rawSegments = useSelectedLayoutSegments();
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);
  const base = `/${workspace.slug}`;

  const items: NavItem[] = [
    {
      icon: null,
      href: `${base}/settings/general`,
      label: "General",
      active: segments.includes("general"),
    },
    {
      icon: null,
      href: `${base}/settings/team`,
      label: "Team",
      active: segments.includes("team"),
    },
    {
      icon: null,
      href: `${base}/settings/root-keys`,
      label: "Root Keys",
      active: segments.includes("root-keys"),
    },
    {
      icon: null,
      href: `${base}/settings/billing`,
      label: "Billing",
      active: segments.includes("billing"),
    },
  ];

  return (
    <div className="flex flex-col gap-2 px-2 py-2 group-data-[collapsible=icon]:px-0">
      <SidebarMenu className="gap-1">
        {items.map((item) => (
          <NavItems key={item.href} item={item} />
        ))}
      </SidebarMenu>
    </div>
  );
}

function Section({
  label,
  last,
  className,
  children,
}: {
  label?: string;
  last?: boolean;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <SidebarGroup
      className={cn(last ? "px-0 py-0" : "border-b border-grayA-4 px-0 py-0", className)}
    >
      {label ? (
        <div className="mb-0.5 px-4 pt-2 text-[12px] font-medium text-gray-11">{label}</div>
      ) : null}
      <div className="px-2">{children}</div>
    </SidebarGroup>
  );
}

function ProjectLeaf({ projectId }: { projectId: string }) {
  const workspace = useWorkspaceNavigation();
  const pathname = usePathname() ?? "";
  const base = `/${workspace.slug}`;
  const projectPath = `${base}/projects/${projectId}`;
  const settingsHref = `${projectPath}/settings`;

  const atOverview = pathname === projectPath || pathname === `${projectPath}/`;
  const atSettings = pathname === settingsHref || pathname.startsWith(`${settingsHref}/`);

  return (
    <div className="flex flex-col gap-2 px-2 py-2 group-data-[collapsible=icon]:px-0">
      <SidebarMenu className="gap-1">
        <NavItems
          item={{
            icon: Grid,
            href: projectPath,
            label: "Overview",
            active: atOverview,
          }}
        />
        <NavItems
          item={{
            icon: Gear,
            href: settingsHref,
            label: "Project Settings",
            active: atSettings,
          }}
        />
      </SidebarMenu>
    </div>
  );
}

function AppLeaf({ projectId, appSlug }: { projectId: string; appSlug: string }) {
  const workspace = useWorkspaceNavigation();
  const pathname = usePathname() ?? "";
  const base = `/${workspace.slug}`;
  const appPath = `${base}/projects/${projectId}/apps/${appSlug}`;

  const isActive = (href: string) => pathname === href || pathname.startsWith(`${href}/`);
  const isOverview = pathname === appPath || pathname === `${appPath}/`;

  const navItems: NavItem[] = [
    { icon: Grid, href: appPath, label: "Overview", active: isOverview },
    {
      icon: Cube,
      href: `${appPath}/deployments`,
      label: "Deployments",
      active: isActive(`${appPath}/deployments`),
    },
    {
      icon: Layers3,
      href: `${appPath}/logs`,
      label: "Logs",
      active: isActive(`${appPath}/logs`),
    },
    {
      icon: ArrowOppositeDirectionY,
      href: `${appPath}/requests`,
      label: "Requests",
      active: isActive(`${appPath}/requests`),
    },
    {
      icon: BracketsSquareDots,
      href: `${appPath}/env-vars`,
      label: "Environment Variables",
      active: isActive(`${appPath}/env-vars`),
    },
    {
      icon: ShieldKey,
      href: `${appPath}/sentinel-policies`,
      label: "Sentinel Policies",
      active: isActive(`${appPath}/sentinel-policies`),
    },
    {
      icon: Gear,
      href: `${appPath}/settings`,
      label: "Settings",
      active: isActive(`${appPath}/settings`),
    },
  ];

  return (
    <div className="flex flex-col gap-2 px-2 py-2 group-data-[collapsible=icon]:px-0">
      <SidebarMenu className="gap-1">
        {navItems.map((item) => (
          <NavItems key={item.href} item={item} />
        ))}
      </SidebarMenu>
    </div>
  );
}
