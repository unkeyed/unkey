"use client";

import { useParams, useSelectedLayoutSegments } from "next/navigation";

export type SectionContext =
  | { type: "workspace" }
  | { type: "settings" }
  | { type: "authorization" }
  | { type: "project"; projectSlug: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string };

export function useSectionContext(): SectionContext {
  const segments = useSelectedLayoutSegments();
  const params = useParams<{
    apiId?: string;
    projectSlug?: string;
    namespaceId?: string;
    identityId?: string;
  }>();

  if (params.projectSlug) {
    return { type: "project", projectSlug: params.projectSlug };
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

  const section = segments[1];
  if (section === "settings") {
    return { type: "settings" };
  }
  if (section === "authorization") {
    return { type: "authorization" };
  }

  return { type: "workspace" };
}
