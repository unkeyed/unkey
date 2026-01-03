"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";

const schema = z.object({
  id: z.string(),
  projectId: z.string(),
  slug: z.string(),
});

export type Environment = z.infer<typeof schema>;

export function createEnvironmentsCollection(projectId: string) {
  if (!projectId) {
    throw new Error("projectId is required to create environments collection");
  }

  return createCollection<Environment>(
    queryCollectionOptions({
      queryClient,
      queryKey: [projectId, "environments"],
      retry: 3,
      queryFn: () => trpcClient.deploy.environment.list.query({ projectId }),
      getKey: (item) => item.id,
      id: `${projectId}-environments`,
    }),
  );
}
