"use client";

import type { Navbar } from "@/components/navigation/navbar";
import { shortenId } from "@/lib/shorten-id";
import { useParams, useSelectedLayoutSegments } from "next/navigation";
import type { ComponentPropsWithoutRef } from "react";

export type QuickNavItem = {
  id: string;
  label: string;
  href: string;
  disabled?: boolean;
  disabledTooltip?: string;
};

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

type SubPage = {
  id: string;
  label: string;
  href: string;
  segment: string | undefined;
  disabled?: boolean;
  disabledTooltip?: string;
};

export const useBreadcrumbConfig = ({
  projectId,
  basePath,
  projects,
  activeProject,
}: {
  projectId: string;
  basePath: string;
  projects: Array<{ id: string; name: string }>;
  activeProject: { id: string; name: string } | undefined;
}): BreadcrumbItem[] => {
  const segments = useSelectedLayoutSegments() ?? [];
  const params = useParams();

  // Find base indices using the segment-based pattern
  const projectsIndex = segments.findIndex((s) => s === "projects");
  const currentSegment = segments.at(projectsIndex + 2); // After [projectId]
  const deploymentId = params?.deploymentId as string | undefined;

  // Sub-pages configuration - matches the existing structure
  const subPages: SubPage[] = [
    {
      id: "overview",
      label: "Overview",
      href: `${basePath}/${projectId}`,
      segment: undefined,
    },
    {
      id: "deployments",
      label: "Deployments",
      href: `${basePath}/${projectId}/deployments`,
      segment: "deployments",
    },
    {
      id: "sentinel-logs",
      label: "Requests",
      href: `${basePath}/${projectId}/sentinel-logs`,
      segment: "sentinel-logs",
    },
    {
      id: "settings",
      label: "Settings",
      href: `${basePath}/${projectId}/settings`,
      segment: "settings",
    },
  ];

  // Determine active subpage based on segment
  const activeSubPage = subPages.find((p) => p.segment === currentSegment) || subPages[0];
  const isOnDeploymentDetail = Boolean(deploymentId);

  // Build breadcrumbs declaratively
  const breadcrumbs: BreadcrumbItem[] = [
    // 1. Projects root
    {
      id: "projects",
      children: "Projects",
      href: basePath,
      shouldRender: true,
      active: false,
      isLast: false,
    },

    // 2. Current project with QuickNav
    {
      id: "project",
      children: activeProject?.name || projectId,
      href: `${basePath}/${projectId}`,
      isIdentifier: true,
      shouldRender: true,
      active: false,
      isLast: false,
      noop: true,
      className: "flex",
      quickNavConfig: {
        items: projects.map((project) => ({
          id: project.id,
          label: project.name,
          href: `${basePath}/${project.id}`,
        })),
        shortcutKey: "N",
      },
    },

    // 3. Sub-page with QuickNav (Overview, Deployments, etc.)
    {
      id: "subpage",
      children: isOnDeploymentDetail ? "Deployments" : activeSubPage.label,
      href: isOnDeploymentDetail ? `${basePath}/${projectId}/deployments` : activeSubPage.href,
      shouldRender: true,
      active: !isOnDeploymentDetail, // Active if not on detail page
      isLast: !isOnDeploymentDetail, // Last if not on detail page
      noop: true,
      quickNavConfig: {
        items: subPages.map((page) => ({
          id: page.id,
          label: page.label,
          href: page.href,
          disabled: page.disabled,
          disabledTooltip: page.disabledTooltip,
        })),
        activeItemId: isOnDeploymentDetail ? "deployments" : undefined,
        shortcutKey: "M",
      },
    },

    // 4. Deployment ID
    {
      id: "deployment-detail",
      children: shortenId(deploymentId || ""),
      href: `${basePath}/${projectId}/deployments/${deploymentId}`,
      isIdentifier: true,
      shouldRender: Boolean(deploymentId),
      active: Boolean(deploymentId),
      isLast: Boolean(deploymentId),
      className: "font-mono",
    },
  ];

  return breadcrumbs.filter((b) => b.shouldRender);
};
