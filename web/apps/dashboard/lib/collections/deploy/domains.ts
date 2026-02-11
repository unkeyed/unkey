"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { parseProjectIdFromWhere, validateProjectIdInQuery } from "./utils";

const schema = z.object({
  id: z.string(),
  fullyQualifiedDomainName: z.string(),
  projectId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
  sticky: z.enum(["none", "branch", "environment", "live"]),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
});

export type Domain = z.infer<typeof schema>;

/**
 * Global domains collection.
 *
 * IMPORTANT: All queries MUST filter by projectId:
 * .where(({ domain }) => eq(domain.projectId, projectId))
 */
export const domains = createCollection<Domain, string>(
  queryCollectionOptions({
    queryClient,
    syncMode: "on-demand",
    queryKey: (opts) => {
      const projectId = parseProjectIdFromWhere(opts.where);
      return projectId ? ["domains", projectId] : ["domains"];
    },
    retry: 3,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      validateProjectIdInQuery(options?.where);
      const projectId = parseProjectIdFromWhere(options?.where);

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      return trpcClient.deploy.domain.list.query({ projectId });
    },
    getKey: (item) => item.id,
    id: "domains",
  }),
);
