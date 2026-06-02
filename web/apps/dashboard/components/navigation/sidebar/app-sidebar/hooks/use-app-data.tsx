"use client";
import type { NavItem } from "@/components/navigation/sidebar/workspace-navigations";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";

/**
 * When viewing a specific app, the sidebar resource is the app, so the parent
 * nav item shows the app name (not the project name). Resolves the app name and
 * relabels the project-resource parent item.
 */
export const useAppData = (baseNavItems: NavItem[], projectSlug?: string, appSlug?: string) => {
  const workspace = useWorkspaceNavigation();

  // app.list filters by project id, so resolve the slug to an id first.
  const { data: projects } = trpc.deploy.project.list.useQuery(undefined, {
    enabled: Boolean(projectSlug),
  });
  const projectId = useMemo(
    () => (projectSlug ? projects?.find((p) => p.slug === projectSlug)?.id : undefined),
    [projects, projectSlug],
  );

  const { data: apps } = trpc.deploy.app.list.useQuery(
    { projectId: projectId ?? "" },
    { enabled: Boolean(projectId) && Boolean(appSlug) },
  );

  const appName = useMemo(
    () => (appSlug ? apps?.find((app) => app.slug === appSlug)?.name : undefined),
    [apps, appSlug],
  );

  const enhancedNavItems = useMemo(() => {
    if (!projectSlug || !appSlug || !appName) {
      return baseNavItems;
    }
    const parentHref = `/${workspace.slug}/projects/${projectSlug}/apps`;
    return baseNavItems.map((item) =>
      item.href === parentHref ? { ...item, label: appName, loading: false } : item,
    );
  }, [baseNavItems, projectSlug, appSlug, appName, workspace.slug]);

  return { enhancedNavItems, appName };
};
