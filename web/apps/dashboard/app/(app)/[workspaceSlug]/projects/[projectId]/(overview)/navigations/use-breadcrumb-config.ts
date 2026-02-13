"use client";

import type { Navbar } from "@/components/navigation/navbar";
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
  const breadcrumbs: BreadcrumbItem[] = [
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

  ];

  return breadcrumbs.filter((b) => b.shouldRender);
};
