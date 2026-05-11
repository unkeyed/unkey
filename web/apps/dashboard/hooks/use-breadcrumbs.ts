"use client";

import { useParams } from "next/navigation";

export type BreadcrumbDescriptor =
  | { type: "workspace" }
  | { type: "project"; projectId: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string };

// Derives the breadcrumb chain from the current route's params. The
// TopNav renders one typed crumb component per descriptor; each crumb
// owns its own data fetching so this hook stays a pure function of
// the URL.
export function useBreadcrumbs(): BreadcrumbDescriptor[] {
  const params = useParams<{
    workspaceSlug?: string;
    projectId?: string;
    apiId?: string;
    namespaceId?: string;
  }>();

  const crumbs: BreadcrumbDescriptor[] = [{ type: "workspace" }];
  if (params.projectId) {
    crumbs.push({ type: "project", projectId: params.projectId });
  }
  if (params.apiId) {
    crumbs.push({ type: "api", apiId: params.apiId });
  }
  if (params.namespaceId) {
    crumbs.push({ type: "namespace", namespaceId: params.namespaceId });
  }
  return crumbs;
}
