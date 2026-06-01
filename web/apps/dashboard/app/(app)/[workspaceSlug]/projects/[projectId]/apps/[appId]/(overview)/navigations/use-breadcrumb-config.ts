"use client";

import type { QuickNavItem } from "@/components/navbar-popover";
import type { Navbar } from "@/components/navigation/navbar";
import { shortenId } from "@/lib/shorten-id";
import { useParams, useSelectedLayoutSegments } from "next/navigation";
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
  appId,
  basePath,
  projects,
  activeProject,
}: {
  projectId: string;
  appId: string;
  basePath: string;
  projects: Array<{ id: string; name: string }>;
  activeProject: { id: string; name: string } | undefined;
}): BreadcrumbItem[] => {
  const segments = useSelectedLayoutSegments() ?? [];
  const params = useParams();
  const deploymentId = params?.deploymentId as string | undefined;

  // All tabs live under the app, e.g. /projects/[projectId]/apps/[appId]/deployments
  const appBase = `${basePath}/${projectId}/apps/${appId}`;

  // Sub-pages configuration - matches the existing structure
  const subPages: SubPage[] = [
    {
      id: "deployments",
      label: "Deployments",
      href: `${appBase}/deployments`,
      segment: "deployments",
    },
    {
      id: "requests",
      label: "Requests",
      href: `${appBase}/requests`,
      segment: "requests",
    },
    {
      id: "logs",
      label: "Logs",
      href: `${appBase}/logs`,
      segment: "logs",
    },
    {
      id: "env-vars",
      label: "Environment Variables",
      href: `${appBase}/env-vars`,
      segment: "env-vars",
    },
    {
      id: "sentinel-policies",
      label: "Sentinel Policies",
      href: `${appBase}/sentinel-policies`,
      segment: "sentinel-policies",
    },
    {
      id: "settings",
      label: "Settings",
      href: `${appBase}/settings`,
      segment: "settings",
    },
    {
      id: "openapi-diff",
      label: "OpenAPI Diff",
      href: `${appBase}/openapi-diff`,
      segment: "openapi-diff",
    },
  ];

  // Determine active subpage by matching any known tab segment in the path.
  const activeSubPage =
    subPages.find((p) => p.segment !== undefined && segments.includes(p.segment)) || subPages[0];
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
      href: isOnDeploymentDetail ? `${appBase}/deployments` : activeSubPage.href,
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
      href: `${appBase}/deployments/${deploymentId}`,
      shouldRender: Boolean(deploymentId),
      active: Boolean(deploymentId),
      isLast: Boolean(deploymentId),
    },
  ];

  return breadcrumbs.filter((b) => b.shouldRender);
};
