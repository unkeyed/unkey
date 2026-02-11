"use client";

import type { QuickNavItem } from "@/components/navbar-popover";
import type { Navbar } from "@/components/navigation/navbar";
import { shortenId } from "@/lib/shorten-id";
import { useParams, useSelectedLayoutSegments } from "next/navigation";
import { useMemo } from "react";
import type { ComponentPropsWithoutRef } from "react";

export type BreadcrumbItem = ComponentPropsWithoutRef<typeof Navbar.Breadcrumbs.Link> & {
  /** Unique identifier for the breadcrumb item */
  id: string;
  /** Internal: determines if this breadcrumb should be rendered */
  shouldRender: boolean;
  /** Optional QuickNav dropdown configuration */
  quickNavConfig?: {
    items: QuickNavItem[];
    activeItemId?: string;
    shortcutKey?: string;
  };
};

export function useDeploymentBreadcrumbConfig(): BreadcrumbItem[] {
  const params = useParams();
  const segments = useSelectedLayoutSegments();

  const workspaceSlug = params.workspaceSlug as string;
  const projectId = params.projectId as string;
  const deploymentId = params.deploymentId as string;

  // Detect current tab from segments
  const currentTab = segments.includes("network")
    ? "network"
    : segments.includes("logs")
      ? "logs"
      : "overview";

  return useMemo(() => {
    const basePath = `/${workspaceSlug}/projects/${projectId}`;

    // Deployment tabs for QuickNav
    const deploymentTabs: QuickNavItem[] = [
      {
        id: "overview",
        label: "Overview",
        href: `${basePath}/deployments/${deploymentId}`,
      },
      {
        id: "logs",
        label: "Logs",
        href: `${basePath}/deployments/${deploymentId}/logs`,
      },
      {
        id: "network",
        label: "Network",
        href: `${basePath}/deployments/${deploymentId}/network`,
      },
    ];

    return [
      {
        id: "projects",
        href: `/${workspaceSlug}/projects`,
        children: "Projects",
        shouldRender: true,
        active: false,
        isLast: false,
      },
      {
        id: "project",
        href: `${basePath}`,
        children: projectId,
        shouldRender: true,
        active: false,
        isLast: false,
      },
      {
        id: "deployments",
        href: `${basePath}/deployments`,
        children: "Deployments",
        shouldRender: true,
        active: false,
        isLast: false,
      },
      {
        id: "deployment",
        href: `${basePath}/deployments/${deploymentId}`,
        children: shortenId(deploymentId),
        isIdentifier: true,
        shouldRender: true,
        active: false,
        isLast: false,
      },
      {
        id: "deployment-tab",
        href: "#",
        noop: true,
        active: true,
        children:
          currentTab === "overview" ? "Overview" : currentTab === "logs" ? "Logs" : "Network",
        shouldRender: true,
        isLast: true,
        quickNavConfig: {
          items: deploymentTabs,
          activeItemId: currentTab,
          shortcutKey: "T",
        },
      },
    ];
  }, [workspaceSlug, projectId, deploymentId, currentTab]);
}
