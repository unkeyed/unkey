"use client";

import { useParams, useSelectedLayoutSegments } from "next/navigation";

export type SectionContext =
  | { type: "workspace" }
  | { type: "settings" }
  | { type: "authorization" }
  | { type: "project"; projectId: string }
  | { type: "deployment"; projectId: string; appId: string; deploymentId: string }
  | { type: "keyDetail"; apiId: string; keyAuthId: string; keyId: string }
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string };

export function useSectionContext(): SectionContext {
  const segments = useSelectedLayoutSegments();
  const params = useParams<{
    apiId?: string;
    keyAuthId?: string;
    keyId?: string;
    projectId?: string;
    appId?: string;
    deploymentId?: string;
    namespaceId?: string;
    identityId?: string;
  }>();

  if (params.deploymentId && params.appId && params.projectId) {
    return {
      type: "deployment",
      projectId: params.projectId,
      appId: params.appId,
      deploymentId: params.deploymentId,
    };
  }
  if (params.projectId) {
    return { type: "project", projectId: params.projectId };
  }
  if (params.keyId && params.keyAuthId && params.apiId) {
    return {
      type: "keyDetail",
      apiId: params.apiId,
      keyAuthId: params.keyAuthId,
      keyId: params.keyId,
    };
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
