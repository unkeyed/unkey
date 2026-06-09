"use client";

import { projectsPath } from "@/lib/navigation/routes/projects";
import { useParams } from "next/navigation";
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

  const workspaceHref = resolveWorkspaceHref(workspace.slug, params);
  const crumbs: BreadcrumbDescriptor[] = [{ type: "workspace", href: workspaceHref }];
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
    return `/${slug}/apis`;
  }
  if (params.projectId) {
    return projectsPath({ workspaceSlug: slug });
  }
  if (params.namespaceId) {
    return `/${slug}/ratelimits`;
  }
  if (params.identityId) {
    return `/${slug}/identities`;
  }
  return `/${slug}`;
}
