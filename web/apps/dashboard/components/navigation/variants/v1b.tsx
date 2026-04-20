"use client";

import { NavItems } from "@/components/navigation/sidebar/app-sidebar/components/nav-items";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { WorkspaceSection } from "@/components/navigation/sidebar/workspace-section";
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
import { cn } from "@/lib/utils";
import { Cube, Fingerprint, Gauge, Grid, Layers3, Nodes, ShieldKey } from "@unkey/icons";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";
import { LeafSidebarView } from "./shared/leaf-sidebar-view";
import { UserButtonFooter } from "./shared/user-button-footer";
import { WorkspaceSwitcherTop } from "./shared/workspace-switcher-top";

type Props = React.ComponentProps<typeof Sidebar> & {
  workspace: Workspace & { quotas: Quotas | null };
};

export function V1bVariant(props: Props) {
  const context = useNavigationContext();

  return (
    <Sidebar {...props}>
      <SidebarHeader className="gap-0 p-0">
        <WorkspaceSwitcherTop />
      </SidebarHeader>
      <SidebarContent>
        {context.type === "resource" ? <LeafSidebarView context={context} /> : <UnifiedTree />}
      </SidebarContent>
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

  const homeItem: NavItem = {
    icon: Grid,
    href: `${base}/home`,
    label: "Home",
    active: segments.at(1) === "home",
  };

  const apiKeysItems: NavItem[] = [
    {
      icon: Nodes,
      href: `${base}/apis`,
      label: "APIs",
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
    },
    {
      icon: Layers3,
      href: `${base}/logs`,
      label: "Logs",
      active: segments.at(1) === "logs",
    },
    {
      icon: Fingerprint,
      href: `${base}/identities`,
      label: "Identities",
      active: segments.at(1) === "identities" && !segments.at(2),
    },
  ];

  return (
    <>
      <Section last className="pb-0">
        <SidebarMenu className="gap-1">
          <NavItems item={homeItem} />
        </SidebarMenu>
      </Section>

      <Section label="Deploy">
        <SidebarMenu className="gap-1">
          <NavItems
            item={{
              icon: Cube,
              href: `${base}/projects`,
              label: "Projects",
              active: segments.at(1) === "projects" && !segments.at(2),
            }}
          />
        </SidebarMenu>
      </Section>

      <Section label="API Keys">
        <SidebarMenu className="gap-1">
          {apiKeysItems.map((item) => (
            <NavItems key={item.href} item={item} />
          ))}
        </SidebarMenu>
      </Section>

      <Section label="Workspace" last>
        <WorkspaceSection />
      </Section>
    </>
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
