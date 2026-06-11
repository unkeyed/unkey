"use client";

import type { Navbar } from "@/components/navigation/navbar";
import { routes } from "@/lib/navigation/routes";
import { shortenId } from "@/lib/shorten-id";
import type { Route } from "next";
import { useParams, useSelectedLayoutSegments } from "next/navigation";
import type { ComponentPropsWithoutRef } from "react";

export type BreadcrumbItem = ComponentPropsWithoutRef<typeof Navbar.Breadcrumbs.Link> & {
  /** Unique identifier for the breadcrumb item */
  id: string;
  /** Internal: determines if this breadcrumb should be rendered */
  shouldRender: boolean;
};

type SubPage = {
  id: string;
  label: string;
  href: Route;
  segment: string | undefined;
  disabled?: boolean;
  disabledTooltip?: string;
};

export const useBreadcrumbConfig = ({
  projectId,
  projectName,
  appId,
  workspaceSlug,
}: {
  projectId: string;
  projectName?: string;
  appId: string;
  workspaceSlug: string;
}): BreadcrumbItem[] => {
  const segments = useSelectedLayoutSegments() ?? [];
  const params = useParams();
  const deploymentId = params?.deploymentId as string | undefined;

  // All tabs live under the app, e.g. /projects/[projectId]/apps/[appId]/deployments
  const appScope = { workspaceSlug, projectId, appId };

  // Sub-pages configuration - matches the existing structure
  const subPages: SubPage[] = [
    {
      id: "deployments",
      label: "Deployments",
      href: routes.projects.apps.deployments(appScope),
      segment: "deployments",
    },
    {
      id: "env-vars",
      label: "Environment Variables",
      href: routes.projects.apps.envVars(appScope),
      segment: "env-vars",
    },
    {
      id: "sentinel-policies",
      label: "Sentinel Policies",
      href: routes.projects.apps.sentinelPolicies(appScope),
      segment: "sentinel-policies",
    },
    {
      id: "settings",
      label: "Settings",
      href: routes.projects.apps.settings(appScope),
      segment: "settings",
    },
    {
      id: "openapi-diff",
      label: "OpenAPI Diff",
      href: routes.projects.apps.openapiDiff(appScope),
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
      href: routes.projects.list({ workspaceSlug }),
      shouldRender: true,
      active: false,
      isLast: false,
    },

    // 2. Current project
    {
      id: "project",
      children: projectName || projectId,
      href: routes.projects.detail({ workspaceSlug, projectId }),
      shouldRender: true,
      active: false,
      isLast: false,
      noop: true,
      className: "flex",
    },

    // 3. Sub-page (Overview, Deployments, etc.)
    {
      id: "subpage",
      children: isOnDeploymentDetail ? "Deployments" : activeSubPage.label,
      href: isOnDeploymentDetail ? routes.projects.apps.deployments(appScope) : activeSubPage.href,
      shouldRender: true,
      active: !isOnDeploymentDetail, // Active if not on detail page
      isLast: !isOnDeploymentDetail, // Last if not on detail page
      noop: true,
    },

    // 3. Deployment ID
    {
      id: "deployment-detail",
      children: shortenId(deploymentId || ""),
      href: deploymentId
        ? routes.projects.apps.deployment({ ...appScope, deploymentId })
        : routes.projects.apps.deployments(appScope),
      shouldRender: Boolean(deploymentId),
      active: Boolean(deploymentId),
      isLast: Boolean(deploymentId),
    },
  ];

  return breadcrumbs.filter((b) => b.shouldRender);
};
