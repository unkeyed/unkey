"use client";

import type { QuickNavItem } from "@/components/navbar-popover";
import type { Navbar } from "@/components/navigation/navbar";
import { shortenId } from "@/lib/shorten-id";
import { useParams } from "next/navigation";
import { useMemo } from "react";
import type { ComponentPropsWithoutRef } from "react";
import { useDeployment } from "../../layout-provider";

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

  const workspaceSlug = params.workspaceSlug as string;
  const { deployment } = useDeployment();

  return useMemo(() => {
    const basePath = `/${workspaceSlug}/projects/${deployment.projectId}`;
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
        children: deployment.projectId,
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
        href: `${basePath}/deployments/${deployment.id}`,
        children: shortenId(deployment.id),
        isIdentifier: true,
        shouldRender: true,
        active: false,
        isLast: false,
      },
    ];
  }, [workspaceSlug, deployment.projectId, deployment.id]);
}
