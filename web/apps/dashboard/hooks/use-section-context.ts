"use client";

import { rememberLastProjectId } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";
import { useParams, useSelectedLayoutSegments } from "next/navigation";
import { useEffect } from "react";

export type SectionContext =
  | { type: "workspace" }
  | { type: "settings" }
  | { type: "authorization" }
  | { type: "project"; projectId: string; appId?: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string };

export function useSectionContext(): SectionContext {
  const segments = useSelectedLayoutSegments();
  const params = useParams<{
    apiId?: string;
    projectId?: string;
    appId?: string;
    namespaceId?: string;
    identityId?: string;
  }>();

  // Track the project the user was last inside so project-scoped concepts
  // living at workspace URLs (authorization) can keep the project sidebar.
  const projectId = params.projectId;
  useEffect(() => {
    if (projectId) {
      rememberLastProjectId(projectId);
    }
  }, [projectId]);

  if (params.projectId) {
    return { type: "project", projectId: params.projectId, appId: params.appId };
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
