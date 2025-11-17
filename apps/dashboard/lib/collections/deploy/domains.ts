"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

const schema = z.object({
  id: z.string(),
  hostname: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
  sticky: z.enum(["branch", "environment", "live"]).nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
});

export type Domain = z.infer<typeof schema>;

export function createDomainsCollection(projectId: string) {
  if (!projectId) {
    throw new Error("projectId is required to create domains collection");
  }

  return createCollection<Domain>(
    queryCollectionOptions({
      queryClient,
      queryKey: [projectId, "domains"],
      retry: 3,
      queryFn: () => trpcClient.deploy.domain.list.query({ projectId }),
      getKey: (item) => item.id,
      id: `${projectId}-domains`,
    })
  );
}
