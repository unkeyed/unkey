"use client";

import { routes } from "@/lib/navigation/routes";
import { Key2 } from "@unkey/icons";
import type { IconProps } from "@unkey/icons";
import { useParams, usePathname } from "next/navigation";
import type { ComponentType } from "react";
import { useWorkspaceNavigation } from "./use-workspace-navigation";

export type BreadcrumbDescriptor =
  | { type: "workspace"; href: string }
  | { type: "project"; projectId: string }
  | { type: "app"; projectId: string; appId: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string }
  | { type: "static"; label: string; href: string; icon?: ComponentType<IconProps> };

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

  const workspaceHref = resolveWorkspaceHref(workspace.slug, params);
  const crumbs: BreadcrumbDescriptor[] = [{ type: "workspace", href: workspaceHref }];
  if (pathname === routes.settings.rootKeyNew({ workspaceSlug: workspace.slug })) {
    crumbs.push({
      type: "static",
      label: "Root keys",
      href: routes.settings.rootKeys({ workspaceSlug: workspace.slug }),
      icon: Key2,
    });
    crumbs.push({
      type: "static",
      label: "New root key",
      href: routes.settings.rootKeyNew({ workspaceSlug: workspace.slug }),
    });
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
