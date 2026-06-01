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
export const useAppData = (baseNavItems: NavItem[], projectId?: string, appId?: string) => {
  const workspace = useWorkspaceNavigation();
  const { data: apps } = trpc.deploy.app.list.useQuery(
    { projectId: projectId ?? "" },
    { enabled: Boolean(projectId) && Boolean(appId) },
  );

  const appName = useMemo(
    () => (appId ? apps?.find((app) => app.id === appId)?.name : undefined),
    [apps, appId],
  );

  const enhancedNavItems = useMemo(() => {
    if (!projectId || !appId || !appName) {
      return baseNavItems;
    }
    const parentHref = `/${workspace.slug}/projects/${projectId}`;
    return baseNavItems.map((item) =>
      item.href === parentHref ? { ...item, label: appName, loading: false } : item,
    );
  }, [baseNavItems, projectId, appId, appName, workspace.slug]);

  return { enhancedNavItems, appName };
};
