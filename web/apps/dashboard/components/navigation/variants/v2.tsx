"use client";

import { NavItems } from "@/components/navigation/sidebar/app-sidebar/components/nav-items";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
} from "@/components/ui/sidebar";
import { useNavigationContext } from "@/hooks/use-navigation-context";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Quotas, Workspace } from "@/lib/db";
import { KEYSPACE_LABEL } from "@/lib/keyspace-label";
import { cn } from "@/lib/utils";
import {
  ChevronLeft,
  ChevronRight,
  Coins,
  Cube,
  Fingerprint,
  Gauge,
  Gear,
  InputSearch,
  Layers3,
  Nodes,
  ShieldKey,
} from "@unkey/icons";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";
import { LeafSidebarView } from "./shared/leaf-sidebar-view";
import { UserButtonFooter } from "./shared/user-button-footer";
import { WorkspaceSwitcherTop } from "./shared/workspace-switcher-top";

type Props = React.ComponentProps<typeof Sidebar> & {
  workspace: Workspace & { quotas: Quotas | null };
};

/**
 * v2 — static sidebar with Deploy-primacy framing.
 *
 * Reflects Andreas's reframe: Deploy is the product; keys, ratelimits,
 * authorization, identities, logs are features. Strict no-dynamic-content
 * rule (no inline project/API lists). Home is dropped — workspace root
 * routes straight to /projects (handled in `page.tsx`).
 */
export function V2Variant(props: Props) {
  const context = useNavigationContext();
  const rawSegments = useSelectedLayoutSegments();
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);
  const isOnSettings = segments.at(1) === "settings";
  const isOnAuthorization = segments.at(1) === "authorization";

  const content = (() => {
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
    <Sidebar {...props}>
      <SidebarHeader className="gap-0 p-0">
        <WorkspaceSwitcherTop />
      </SidebarHeader>
      <SidebarContent>{content}</SidebarContent>
      <SidebarFooter className="mx-0 gap-0 border-t-0 p-0">
        <UserButtonFooter />
      </SidebarFooter>
    </Sidebar>
  );
}

function UnifiedTree() {
  const workspace = useWorkspaceNavigation();
  const rawSegments = useSelectedLayoutSegments();
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);
  const base = `/${workspace.slug}`;

  const drillTag = <ChevronRight className="size-2.5 text-gray-9" iconSize="sm-thin" />;

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

  // Inline workspace items so they share the same SidebarMenu shell as
  // Features and Deploy. Mirrors `WorkspaceSection`'s contents but skips
  // its extra SidebarGroup wrapper, which was causing the items to appear
  // visually nested compared to Features above.
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
      <Section label="Deploy">
        <SidebarMenu className="gap-1">
          <NavItems item={deployItem} />
        </SidebarMenu>
      </Section>

      <Section label="Features">
        <SidebarMenu className="gap-1">
          {featureItems.map((item) => (
            <NavItems key={item.href} item={item} />
          ))}
        </SidebarMenu>
      </Section>

      <Section label="Workspace" last>
        <SidebarMenu className="gap-1">
          {workspaceItems.map((item) => (
            <NavItems key={item.href} item={item} />
          ))}
        </SidebarMenu>
      </Section>
    </>
  );
}

/**
 * Leaf view for `/[ws]/authorization/*` routes. Authorization isn't a
 * "resource" in `useNavigationContext`'s model (it's workspace-level),
 * so we mirror the SettingsLeaf pattern here.
 */
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
    <div className="flex flex-col gap-2 px-2 pt-1">
      <Link
        href={base}
        className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-[12px] font-medium text-gray-11 hover:bg-gray-3 hover:text-gray-12"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
        <span>Workspace</span>
      </Link>
      <SidebarMenu className="gap-1">
        {items.map((item) => (
          <NavItems key={item.href} item={item} />
        ))}
      </SidebarMenu>
    </div>
  );
}

/**
 * Leaf view for `/[ws]/settings/*` routes. Settings isn't a "resource" in
 * `useNavigationContext`'s model, so we special-case it here: back button
 * to the workspace root + the settings sub-nav as flat list items.
 */
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
    <div className="flex flex-col gap-2 px-2 pt-1">
      <Link
        href={base}
        className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-[12px] font-medium text-gray-11 hover:bg-gray-3 hover:text-gray-12"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
        <span>Workspace</span>
      </Link>
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
      className={cn(last ? "px-0 py-2" : "border-b border-grayA-4 px-0 py-2", className)}
    >
      {label ? <div className="mb-1 px-4 text-[12px] font-medium text-gray-11">{label}</div> : null}
      <div className="px-2">{children}</div>
    </SidebarGroup>
  );
}
