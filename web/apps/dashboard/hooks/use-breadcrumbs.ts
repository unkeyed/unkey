"use client";

import { routes } from "@/lib/navigation/routes";
import { useParams, usePathname } from "next/navigation";
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

  // The create-project page has no entity params, so it gets a static crumb.
  const isNewProject = pathname === routes.projects.new({ workspaceSlug: workspace.slug });

  const workspaceHref = isNewProject
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
