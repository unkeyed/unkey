"use client";

import { useFlag } from "@/lib/flags/provider";
import { routes } from "@/lib/navigation/routes";
import { useParams } from "next/navigation";
import { type ProjectResource, useResourceProjectId } from "./use-resource-project-id";
import { useWorkspaceNavigation } from "./use-workspace-navigation";

export type BreadcrumbDescriptor =
  | { type: "workspace"; href: string }
  | { type: "project"; projectId: string }
  | { type: "app"; projectId: string; appId: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string };

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
  const projectsNav = useFlag("projectsNav");
  const ownerProjectId = useResourceProjectId(projectsNav ? resourceFromParams(params) : null);

  const workspaceHref = ownerProjectId
    ? routes.projects.list({ workspaceSlug: workspace.slug })
    : resolveWorkspaceHref(workspace.slug, params);
  const crumbs: BreadcrumbDescriptor[] = [{ type: "workspace", href: workspaceHref }];
  if (params.projectId) {
    crumbs.push({ type: "project", projectId: params.projectId });
  }
  if (ownerProjectId) {
    crumbs.push({ type: "project", projectId: ownerProjectId });
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

function resourceFromParams(params: RouteParams): ProjectResource | null {
  if (params.projectId) {
    return null;
  }
  if (params.apiId) {
    return { type: "api", apiId: params.apiId };
  }
  if (params.namespaceId) {
    return { type: "namespace", namespaceId: params.namespaceId };
  }
  if (params.identityId) {
    return { type: "identity", identityId: params.identityId };
  }
  return null;
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
