"use client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import type { environments as environmentsTable } from "@unkey/db/src/schema";
import { z } from "zod";
import { queryClient, trpcClient } from "../client";
import { parseProjectIdFromWhere, validateProjectIdInQuery } from "./utils";

const kind = z.enum(["production", "preview"] as const satisfies readonly KindColumn[]);

/** Named access to the column values, e.g. `ENVIRONMENT_KIND.production`. */
export const ENVIRONMENT_KIND = kind.enum;
export const ENVIRONMENT_KINDS = kind.options;
export type EnvironmentKind = z.infer<typeof kind>;

const schema = z.object({
  id: z.string(),
  projectId: z.string(),
  slug: z.string(),
  kind,
  appId: z.string(),
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

/**
 * `environments.kind` as the database declares it. The table is imported as a
 * type only, so `satisfies` above breaks the build when the column drops a
 * value, and drizzle stays out of the browser bundle.
 */
type KindColumn = (typeof environmentsTable)["kind"]["enumValues"][number];
