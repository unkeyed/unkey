"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { match } from "@unkey/match";

export type ProjectResource =
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string };

export type ProjectOwner =
  | { state: "loading" }
  | { state: "resolved"; projectId: string }
  | { state: "unknown" };

const loading: ProjectOwner = { state: "loading" };
const unknown: ProjectOwner = { state: "unknown" };

function owner(projectId: string | undefined): ProjectOwner {
  return projectId ? { state: "resolved", projectId } : unknown;
}

export function useResourceProjectId(resource: ProjectResource | null): ProjectOwner | null {
  const apiId = resource?.type === "api" ? resource.apiId : null;
  const namespaceId = resource?.type === "namespace" ? resource.namespaceId : null;
  const identityId = resource?.type === "identity" ? resource.identityId : null;

  const apiQuery = trpc.api.queryApiKeyDetails.useQuery(
    { apiId: apiId ?? "" },
    { enabled: Boolean(apiId) },
  );

  const namespaceQuery = useLiveQuery(
    (q) =>
      namespaceId
        ? q
            .from({ namespace: collection.ratelimitNamespaces })
            .where(({ namespace }) => eq(namespace.id, namespaceId))
        : null,
    [namespaceId],
  );

  const identityQuery = trpc.identity.details.useQuery(
    { identityId: identityId ?? "" },
    { enabled: Boolean(identityId) },
  );

  if (!resource) {
    return null;
  }

  return match(resource)
    .with({ type: "api" }, () => {
      if (apiQuery.isError) {
        return unknown;
      }
      return apiQuery.data ? owner(apiQuery.data.currentApi.projectId) : loading;
    })
    .with({ type: "namespace" }, () => {
      if (namespaceQuery.isError) {
        return unknown;
      }
      return namespaceQuery.isLoading ? loading : owner(namespaceQuery.data?.at(0)?.projectId);
    })
    .with({ type: "identity" }, () => {
      if (identityQuery.isError) {
        return unknown;
      }
      return identityQuery.data ? owner(identityQuery.data.projectId) : loading;
    })
    .exhaustive();
}
