"use client";
import { getUnkeyClient } from "@/lib/unkey-client";
import type { Environment } from "@unkey/api/models/components";
import { queryClient } from "../client";

function appEnvironmentsKey(projectId: string, appId: string) {
  return ["listEnvironments", projectId, appId];
}

export function listAppEnvironments(projectId: string, appId: string): Promise<Environment[]> {
  return queryClient.fetchQuery({
    queryKey: appEnvironmentsKey(projectId, appId),
    queryFn: async () => {
      const { data } = await getUnkeyClient().environments.listEnvironments({
        project: projectId,
        app: appId,
      });
      return data;
    },
    staleTime: 0,
  });
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
