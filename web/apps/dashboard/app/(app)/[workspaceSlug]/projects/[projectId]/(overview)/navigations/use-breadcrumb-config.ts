"use client";

import type { QuickNavItem } from "@/components/navbar-popover";
import type { Navbar } from "@/components/navigation/navbar";
import { useProjectItems } from "@/hooks/use-project-items";
import { type ProjectItem, type ProjectItemType, sortByTypeGroup } from "@/lib/project-items";
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

/**
 * Project-as-container model (Andreas, 2026-04-21): the breadcrumb's
 * third segment is either **Overview** (project root) or the name of an
 * item inside the project (app / database / queue / vault). Items come
 * from `useProjectItems` — localStorage-backed while the backend catches
 * up.
 *
 * Legacy app-level routes (`deployments`, `logs`, `env-vars`,
 * `sentinel-policies`, `settings`, `requests`, `openapi-diff`) still
 * resolve as URLs so pre-existing bookmarks don't 404; the breadcrumb
 * surfaces them with a generic fallback label rather than pretending
 * they're a first-class sibling of the new items.
 */
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
  const projectsIndex = segments.findIndex((s) => s === "projects");
  const typeSegment = segments.at(projectsIndex + 2);
  const slugSegment = segments.at(projectsIndex + 3);
  const appSectionSegment = segments.at(projectsIndex + 4);
  const deploymentId = params?.deploymentId as string | undefined;

  const { items } = useProjectItems(projectId);
  const sortedItems = useMemo(() => sortByTypeGroup(items), [items]);

  const typeForSegment = (seg: string | undefined): ProjectItemType | undefined => {
    switch (seg) {
      case "apps":
        return "app";
      case "databases":
        return "database";
      case "queues":
        return "queue";
      case "vault":
        return "vault";
      default:
        return undefined;
    }
  };

  const currentItemType = typeForSegment(typeSegment);
  const currentItem =
    currentItemType !== undefined && slugSegment
      ? items.find((i) => i.type === currentItemType && i.slug === slugSegment)
      : undefined;

  const overviewHref = `${basePath}/${projectId}`;
  const subpageQuickNav: QuickNavItem[] = [
    {
      id: "overview",
      label: "Overview",
      href: overviewHref,
    },
    ...sortedItems.map((item) => ({
      id: `${item.type}-${item.slug}`,
      label: item.name,
      href: itemHref(basePath, projectId, item),
    })),
  ];

  const activeLabel = currentItem
    ? currentItem.name
    : typeSegment && !currentItemType
      ? legacyLabel(typeSegment)
      : "Overview";

  const activeHref = currentItem
    ? itemHref(basePath, projectId, currentItem)
    : typeSegment && !currentItemType
      ? `${basePath}/${projectId}/${typeSegment}`
      : overviewHref;

  const isOnDeploymentDetail = Boolean(deploymentId);
  const activeQuickNavId = currentItem
    ? `${currentItem.type}-${currentItem.slug}`
    : typeSegment
      ? undefined
      : "overview";

  const breadcrumbs: BreadcrumbItem[] = [
    {
      id: "projects",
      children: "Projects",
      href: basePath,
      shouldRender: true,
      active: false,
      isLast: false,
    },
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
    {
      id: "subpage",
      children: isOnDeploymentDetail ? "Deployments" : activeLabel,
      href: isOnDeploymentDetail ? `${basePath}/${projectId}/deployments` : activeHref,
      shouldRender: true,
      active: !isOnDeploymentDetail && !appSectionSegment,
      isLast: !isOnDeploymentDetail && !appSectionSegment,
      noop: true,
      quickNavConfig: {
        items: subpageQuickNav,
        activeItemId: isOnDeploymentDetail ? undefined : activeQuickNavId,
        shortcutKey: "M",
      },
    },
    {
      id: "app-section",
      children: appSectionSegment ? legacyLabel(appSectionSegment) : "",
      href:
        currentItem && appSectionSegment
          ? `${itemHref(basePath, projectId, currentItem)}/${appSectionSegment}`
          : "",
      shouldRender: Boolean(appSectionSegment && currentItemType === "app"),
      active: Boolean(appSectionSegment),
      isLast: Boolean(appSectionSegment) && !isOnDeploymentDetail,
    },
    {
      id: "deployment-detail",
      children: shortenId(deploymentId || ""),
      href: `${basePath}/${projectId}/deployments/${deploymentId}`,
      shouldRender: Boolean(deploymentId),
      active: Boolean(deploymentId),
      isLast: Boolean(deploymentId),
    },
  ];

  return breadcrumbs.filter((b) => b.shouldRender);
};

function itemHref(basePath: string, projectId: string, item: ProjectItem): string {
  const base = `${basePath}/${projectId}`;
  switch (item.type) {
    case "app":
      return `${base}/apps/${item.slug}`;
    case "database":
      return `${base}/databases/${item.slug}`;
    case "queue":
      return `${base}/queues/${item.slug}`;
    case "vault":
      return `${base}/vault/${item.slug}`;
  }
}

function legacyLabel(segment: string): string {
  switch (segment) {
    case "deployments":
      return "Deployments";
    case "requests":
      return "Requests";
    case "logs":
      return "Logs";
    case "env-vars":
      return "Environment Variables";
    case "sentinel-policies":
      return "Sentinel Policies";
    case "settings":
      return "Settings";
    case "openapi-diff":
      return "OpenAPI Diff";
    default:
      return segment.replace(/[-_]/g, " ");
  }
}
