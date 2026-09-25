"use client";
import { getUnkeyClient } from "@/lib/unkey-client";
import { parseLoadSubsetOptions, queryCollectionOptions } from "@tanstack/query-db-collection";
import { createCollection } from "@tanstack/react-db";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import type { environments as environmentsTable } from "@unkey/db/src/schema";
import { z } from "zod";
import { queryClient } from "../client";
import { extractStringFilter, extractStringsFilter } from "./utils";

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
 * Global environments collection. The API lists the environments of one app,
 * so every query names its apps.
 *
 * IMPORTANT: All queries MUST filter by projectId and by appId with eq or inArray:
 * .where(({ env }) => and(eq(env.projectId, projectId), inArray(env.appId, appIds)))
 *
 * Do not reach it through the nullable side of an outer join. TanStack DB does
 * not push that side's where clause into the load, so the projectId is lost.
 */
export const environments = createCollection<Environment, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const { filters } = parseLoadSubsetOptions(opts);
      const projectId = extractStringFilter(filters, "projectId");
      const appIds = extractStringsFilter(filters, "appId").sort();
      return projectId ? ["environments", projectId, ...appIds] : ["environments"];
    },
    syncMode: "on-demand",
    retry: 3,
    queryFn: async (ctx) => {
      const { filters } = parseLoadSubsetOptions(ctx.meta?.loadSubsetOptions);
      const projectId = extractStringFilter(filters, "projectId");
      const appIds = extractStringsFilter(filters, "appId");

      if (!projectId || appIds.length === 0) {
        throw new Error(
          "Query must include eq(collection.projectId, projectId) and an eq or inArray constraint on collection.appId",
        );
      }

      const perApp = await Promise.all(
        appIds.map(async (appId) => {
          try {
            const { data } = await getUnkeyClient().environments.listEnvironments({
              project: projectId,
              app: appId,
            });
            return data.map((e) => ({ id: e.id, projectId, appId, slug: e.slug, kind: e.kind }));
          } catch (error) {
            // A deleted app or one without read permission must not hide the other apps' environments
            if (error instanceof NotFoundErrorResponse) {
              return [];
            }
            throw error;
          }
        }),
      );

      return perApp.flat();
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
