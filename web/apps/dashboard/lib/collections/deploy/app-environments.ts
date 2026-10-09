"use client";
import { getUnkeyClient } from "@/lib/unkey-client";
import type { Environment } from "@unkey/api/models/components";
import { queryClient } from "../client";

const writes = new Map<string, number>();

export function listAppEnvironments(projectId: string, appId: string): Promise<Environment[]> {
  return queryClient.fetchQuery({
    // An overlapping call joins the request in flight, even one that started
    // before a write, so each write moves later calls to a new key.
    queryKey: ["listEnvironments", projectId, appId, writes.get(`${projectId}:${appId}`) ?? 0],
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

export function markAppEnvironmentsChanged(projectId: string, appId: string): void {
  const app = `${projectId}:${appId}`;
  writes.set(app, (writes.get(app) ?? 0) + 1);
}
