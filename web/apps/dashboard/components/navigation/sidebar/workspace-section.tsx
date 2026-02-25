"use client";

import { SidebarGroup, SidebarMenu } from "@/components/ui/sidebar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Coins, Gear, InputSearch } from "@unkey/icons";
import { useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";
import { NavItems } from "./app-sidebar/components/nav-items";
import type { NavItem } from "./workspace-navigations";

/**
 * Workspace section component that contains workspace-level navigation items.
 * The workspace switcher is now rendered separately at the bottom of the sidebar.
 */
export function WorkspaceSection() {
  const rawSegments = useSelectedLayoutSegments();
  const workspace = useWorkspaceNavigation();

  // Memoize segments to prevent unnecessary re-renders
  const segments = useMemo(() => rawSegments ?? [], [rawSegments]);

  const workspaceItems = useMemo<NavItem[]>(() => {
    const basePath = `/${workspace.slug}`;

    return [
      {
        icon: InputSearch,
        href: `${basePath}/audit`,
        label: "Audit Log",
        active: segments.at(1) === "audit",
      },
      {
        icon: Coins,
        href: `${basePath}/portal`, // url TBD
        label: "Customer Billing",
        active: segments.at(1) === "portal",
        hidden: !workspace.betaFeatures?.portal,
      },
      {
        icon: Gear,
        href: `${basePath}/settings/general`,
        label: "Workspace Settings",
        active: segments.at(1) === "settings",
        items: [
          {
            icon: null,
            href: `${basePath}/settings/general`,
            label: "General",
            active: segments.includes("general"),
          },
          {
            icon: null,
            href: `${basePath}/settings/team`,
            label: "Team",
            active: segments.includes("team"),
          },
          {
            icon: null,
            href: `${basePath}/settings/root-keys`,
            label: "Root Keys",
            active: segments.includes("root-keys"),
          },
          {
            icon: null,
            href: `${basePath}/settings/billing`,
            label: "Billing",
            active: segments.includes("billing"),
          },
        ],
      },
    ].filter((item) => !item.hidden);
  }, [segments, workspace.slug, workspace.betaFeatures?.portal]);

  return (
    <SidebarGroup>
      {/* Workspace-level navigation items */}
      <SidebarMenu className="gap-2">
        {workspaceItems.map((item) => (
          <NavItems key={item.href} item={item} />
        ))}
      </SidebarMenu>
    </SidebarGroup>
  );
}
