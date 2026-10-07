"use client";

import { useParams, useSelectedLayoutSegments } from "next/navigation";

export type SectionContext =
  | { type: "workspace" }
  | { type: "account" }
  | { type: "settings" }
  | { type: "authorization" }
  | { type: "project"; projectId: string; appId?: string }
  | { type: "api"; apiId: string; projectId?: string }
  | { type: "namespace"; namespaceId: string; projectId?: string }
  | { type: "identity"; identityId: string; projectId?: string };

export function useSectionContext(): SectionContext {
  const segments = useSelectedLayoutSegments();
  const params = useParams<{
    apiId?: string;
    projectId?: string;
    appId?: string;
    namespaceId?: string;
    identityId?: string;
  }>();

  // Resource ids win over projectId so project-mounted detail pages get the resource rail.
  if (params.apiId) {
    return { type: "api", apiId: params.apiId, projectId: params.projectId };
  }
  if (params.namespaceId) {
    return { type: "namespace", namespaceId: params.namespaceId, projectId: params.projectId };
  }
  if (params.identityId) {
    return { type: "identity", identityId: params.identityId, projectId: params.projectId };
  }
  if (params.projectId) {
    return { type: "project", projectId: params.projectId, appId: params.appId };
  }

  const section = segments[1];
  if (section === "settings") {
    return { type: "settings" };
  }
  if (section === "authorization") {
    return { type: "authorization" };
  }
  if (section === "account") {
    return { type: "account" };
  }

  return { type: "workspace" };
}
