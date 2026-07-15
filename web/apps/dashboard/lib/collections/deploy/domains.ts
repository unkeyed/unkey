"use client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { validateProjectIdInQuery } from "./utils";

const schema = z.object({
  id: z.string(),
  fullyQualifiedDomainName: z.string(),
  projectId: z.string(),
  appId: z.string(),
  deploymentId: z.string(),
  environmentId: z.string(),
  sticky: z.enum(["none", "branch", "environment", "live", "deployment"]),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
});

export type Domain = z.infer<typeof schema>;

type ParsedFilter = { field: Array<string | number>; operator: string; value?: unknown };

function extractStringFilter(filters: ParsedFilter[], fieldName: string, operator: string) {
  const value = filters.find((f) => f.field.at(-1) === fieldName && f.operator === operator)?.value;
  return typeof value === "string" ? value : undefined;
}

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
      const { filters } = parseLoadSubsetOptions(opts);
      const projectId = extractStringFilter(filters, "projectId", "eq");
      const appId = extractStringFilter(filters, "appId", "eq");
      if (!projectId) {
        return ["domains"];
      }
      return appId ? ["domains", projectId, appId] : ["domains", projectId];
    },
    retry: 3,
    queryFn: async (ctx) => {
      const options = ctx.meta?.loadSubsetOptions;

      validateProjectIdInQuery(options?.where);
      const { filters } = parseLoadSubsetOptions(options);
      const projectId = extractStringFilter(filters, "projectId", "eq");
      const appId = extractStringFilter(filters, "appId", "eq");

      if (!projectId) {
        throw new Error("Query must include eq(collection.projectId, projectId) constraint");
      }

      return trpcClient.deploy.domain.list.query({
        projectId,
        ...(appId !== undefined && { appId }),
      });
    },
    getKey: (item) => item.id,
    id: "domains",
  }),
);
