"use client";

import {
  findProjectIdForResource,
  readLastProjectId,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";
import { routes } from "@/lib/navigation/routes";
import { useParams, usePathname } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useWorkspaceNavigation } from "./use-workspace-navigation";

export type BreadcrumbDescriptor =
  | { type: "workspace"; href: string }
  | { type: "project"; projectId: string }
  | { type: "app"; projectId: string; appId: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string }
  | { type: "label"; label: string };

type RouteParams = {
  projectId?: string;
  appId?: string;
  apiId?: string;
  namespaceId?: string;
  identityId?: string;
};

export function useBreadcrumbs(): BreadcrumbDescriptor[] {
  const workspace = useWorkspaceNavigation();
  const params = useParams<RouteParams>();
  const pathname = usePathname();

  // Keyspaces and ratelimits live inside projects now, but their pages mount
  // at workspace-scoped URLs — resolve the owning (prototype) project so the
  // breadcrumb still shows where the resource sits.
  const resourceId = params.apiId ?? params.namespaceId;
  const resourceProjectId = useMemo(
    () => (resourceId ? findProjectIdForResource(resourceId) : null),
    [resourceId],
  );

  // Authorization is project-scoped conceptually but lives at workspace URLs;
  // surface the project the user was last inside (matches the sidebar).
  const isAuthorization = pathname.startsWith(`/${workspace.slug}/authorization`);
  const [lastProjectId, setLastProjectId] = useState<string | null>(null);
  useEffect(() => {
    if (isAuthorization) {
      setLastProjectId(readLastProjectId());
    }
  }, [isAuthorization]);

  // The create-project page has no entity params, so it gets a static crumb.
  const isNewProject = pathname === routes.projects.new({ workspaceSlug: workspace.slug });

  const workspaceHref =
    isNewProject || resourceProjectId || (isAuthorization && lastProjectId)
      ? routes.projects.list({ workspaceSlug: workspace.slug })
      : resolveWorkspaceHref(workspace.slug, params);
  const crumbs: BreadcrumbDescriptor[] = [{ type: "workspace", href: workspaceHref }];
  if (isNewProject) {
    crumbs.push({ type: "label", label: "New project" });
  }
  if (params.projectId) {
    crumbs.push({ type: "project", projectId: params.projectId });
  }
  if (params.projectId && params.appId) {
    crumbs.push({ type: "app", projectId: params.projectId, appId: params.appId });
  }
  if (!params.projectId && resourceProjectId) {
    crumbs.push({ type: "project", projectId: resourceProjectId });
  }
  if (isAuthorization && lastProjectId) {
    crumbs.push({ type: "project", projectId: lastProjectId });
    crumbs.push({ type: "label", label: "Authorization" });
  }
  if (params.apiId) {
    crumbs.push({ type: "api", apiId: params.apiId });
  }
  if (params.namespaceId) {
    crumbs.push({ type: "namespace", namespaceId: params.namespaceId });
  }
  if (params.identityId) {
    crumbs.push({ type: "identity", identityId: params.identityId });
  }
  return crumbs;
}

function resolveWorkspaceHref(slug: string, params: RouteParams): string {
  if (params.apiId) {
    return routes.apis.list({ workspaceSlug: slug });
  }
  if (params.projectId) {
    return routes.projects.list({ workspaceSlug: slug });
  }
  if (params.namespaceId) {
    return routes.ratelimits.list({ workspaceSlug: slug });
  }
  if (params.identityId) {
    return `/${slug}/identities`;
  }
  return `/${slug}`;
}
