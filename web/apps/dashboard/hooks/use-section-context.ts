"use client";

import { useParams, useSelectedLayoutSegments } from "next/navigation";

// Drives which leaf the contextual sidebar renders. Pure function of
// the URL — no localStorage, no product split. PR 4 deletes the legacy
// useNavigationContext that this replaces.
export type SectionContext =
  | { type: "workspace" }
  | { type: "settings" }
  | { type: "authorization" }
  | { type: "project"; projectId: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string };

export function useSectionContext(): SectionContext {
  const segments = useSelectedLayoutSegments();
  const params = useParams<{
    apiId?: string;
    projectId?: string;
    namespaceId?: string;
  }>();

  if (params.projectId) {
    return { type: "project", projectId: params.projectId };
  }
  if (params.apiId) {
    return { type: "api", apiId: params.apiId };
  }
  if (params.namespaceId) {
    return { type: "namespace", namespaceId: params.namespaceId };
  }

  // segments[0] is the workspace slug from [workspaceSlug]; the section name
  // is the next segment.
  const section = segments[1];
  if (section === "settings") {
    return { type: "settings" };
  }
  if (section === "authorization") {
    return { type: "authorization" };
  }

  return { type: "workspace" };
}
