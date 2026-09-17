"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { projectDisplayName } from "@/lib/collections/deploy/projects";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { useMemo } from "react";
import type { LaunchpadModel, LaunchpadRow } from "./types";

export function useLaunchpad(windowHours = 24): LaunchpadModel {
  const workspace = useWorkspaceNavigation();
  const workspaceSlug = workspace.slug;
  const query = trpc.launchpad.overview.useQuery({ windowHours });

  return useMemo(() => {
    const data = query.data;
    const projectNames = new Map(
      (data?.projects ?? []).map((project) => [
        project.id,
        projectDisplayName(project, workspace.name),
      ]),
    );

    const rows: LaunchpadRow[] = (data?.items ?? []).map((item) => ({
      id: item.id,
      name: item.name,
      kind: item.kind,
      keyCount: item.keyCount,
      total: item.total,
      failed: item.failed,
      buckets: item.buckets,
      projectName: item.projectId ? (projectNames.get(item.projectId) ?? null) : null,
      href:
        item.kind === "keyspace"
          ? routes.apis.detail({ workspaceSlug, apiId: item.id })
          : routes.ratelimits.detail({ workspaceSlug, namespaceId: item.id }),
    }));

    return {
      isLoading: query.isLoading,
      rows,
      keyspaces: rows.filter((row) => row.kind === "keyspace"),
      ratelimits: rows.filter((row) => row.kind === "ratelimit"),
      identityCount: data?.identityCount ?? 0,
      identitiesHref: routes.identities.list({ workspaceSlug }),
      keyspacesHref: routes.apis.list({ workspaceSlug }),
      ratelimitsHref: routes.ratelimits.list({ workspaceSlug }),
      projectCount: data?.projects.length ?? 0,
      windowHours: data?.windowHours ?? windowHours,
      isEmpty: !query.isLoading && rows.length === 0 && (data?.identityCount ?? 0) === 0,
    };
  }, [query.data, query.isLoading, workspaceSlug, workspace.name, windowHours]);
}
