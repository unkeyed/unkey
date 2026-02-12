"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { parseProjectIdFromWhere, validateProjectIdInQuery } from "./utils";

const schema = z.object({
  id: z.string(),
  projectId: z.string(),
  slug: z.string(),
});

export type Environment = z.infer<typeof schema>;

/**
 * Global environments collection.
 *
 * IMPORTANT: All queries MUST filter by projectId:
 * .where(({ environment }) => eq(environment.projectId, projectId))
 */
export const environments = createCollection<Environment, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const projectId = parseProjectIdFromWhere(opts.where);
      return projectId ? ["environments", projectId] : ["environments"];
    },
    syncMode: "on-demand",
    retry: 3,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      validateProjectIdInQuery(options?.where);
      const projectId = parseProjectIdFromWhere(options?.where);

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      return trpcClient.deploy.environment.list.query({ projectId });
    },
    getKey: (item) => item.id,
    id: "environments",
  }),
);
