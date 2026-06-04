"use client";

import { useParams } from "next/navigation";
import { useWorkspaceNavigation } from "./use-workspace-navigation";

export type BreadcrumbDescriptor =
  | { type: "workspace"; href: string }
  | { type: "project"; projectSlug: string }
  | { type: "app"; projectSlug: string; appSlug: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string };

type RouteParams = {
  projectSlug?: string;
  appSlug?: string;
  apiId?: string;
  namespaceId?: string;
  identityId?: string;
};

export function useBreadcrumbs(): BreadcrumbDescriptor[] {
  const workspace = useWorkspaceNavigation();
  const params = useParams<RouteParams>();

  const workspaceHref = resolveWorkspaceHref(workspace.slug, params);
  const crumbs: BreadcrumbDescriptor[] = [{ type: "workspace", href: workspaceHref }];
  if (params.projectSlug) {
    crumbs.push({ type: "project", projectSlug: params.projectSlug });
  }
  if (params.projectSlug && params.appSlug) {
    crumbs.push({ type: "app", projectSlug: params.projectSlug, appSlug: params.appSlug });
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
  if (params.projectSlug) {
    return `/${slug}/projects`;
  }
  if (params.namespaceId) {
    return `/${slug}/ratelimits`;
  }
  if (params.identityId) {
    return `/${slug}/identities`;
  }
  return `/${slug}`;
}
