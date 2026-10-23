"use client";
import { getUnkeyClient } from "@/lib/unkey-client";
import type { Environment } from "@unkey/api/models/components";
import { queryClient } from "../client";

function appEnvironmentsKey(projectId: string, appId: string) {
  return ["listEnvironments", projectId, appId];
}

function appEnvironmentsQuery(projectId: string, appId: string) {
  return {
    queryKey: appEnvironmentsKey(projectId, appId),
    queryFn: async () => {
      const { data } = await getUnkeyClient().environments.listEnvironments({
        project: projectId,
        app: appId,
      });
      return data;
    },
  };
}

export function listAppEnvironments(projectId: string, appId: string): Promise<Environment[]> {
  return queryClient.fetchQuery({ ...appEnvironmentsQuery(projectId, appId), staleTime: 0 });
}

export function cachedAppEnvironmentIds(projectId: string, appId: string): Promise<string[]> {
  return queryClient
    .ensureQueryData(appEnvironmentsQuery(projectId, appId))
    .then((environments) => environments.map((environment) => environment.id));
}

/**
 * Call after a write. fetchQuery joins a request in flight, even one that started
 * before the write; refetching cancels that request and starts one that sees the write.
 */
export function refetchAppEnvironments(projectId: string, appId: string): Promise<void> {
  return queryClient.refetchQueries(
    { queryKey: appEnvironmentsKey(projectId, appId), exact: true },
    { cancelRefetch: true },
  );
}
