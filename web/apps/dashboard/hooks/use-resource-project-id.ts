"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { match } from "@unkey/match";

export type ProjectResource =
  | { type: "api"; apiId: string }
  | { type: "namespace"; namespaceId: string }
  | { type: "identity"; identityId: string };

export function useResourceProjectId(resource: ProjectResource | null): string | null {
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
    .with({ type: "api" }, () => apiQuery.data?.currentApi.projectId ?? null)
    .with({ type: "namespace" }, () => namespaceQuery.data?.at(0)?.projectId ?? null)
    .with({ type: "identity" }, () => identityQuery.data?.projectId ?? null)
    .exhaustive();
}
