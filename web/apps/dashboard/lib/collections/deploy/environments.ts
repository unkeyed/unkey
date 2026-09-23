"use client";
import { getUnkeyClient } from "@/lib/unkey-client";
import { queryCollectionOptions } from "@tanstack/query-db-collection";
import {
  type IR,
  type Ref,
  and,
  createCollection,
  eq,
  inArray,
  or,
  parseWhereExpression,
} from "@tanstack/react-db";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";
import type { environments as environmentsTable } from "@unkey/db/src/schema";
import { z } from "zod";
import { queryClient } from "../client";

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
 * so every query names its apps by project.
 *
 * IMPORTANT: All queries MUST filter by projectId and by appId with eq or inArray:
 * .where(({ env }) => and(eq(env.projectId, projectId), inArray(env.appId, appIds)))
 *
 * For apps in more than one project, build the where with `inProjectApps`:
 * .where(({ env }) => inProjectApps(env, [
 *   { id: "proj_1", apps: [{ id: "app_a" }, { id: "app_b" }] },
 *   { id: "proj_2", apps: [{ id: "app_c" }] },
 * ]))
 * That is or(and(eq(projectId, "proj_1"), inArray(appId, ["app_a", "app_b"])),
 * and(eq(projectId, "proj_2"), inArray(appId, ["app_c"]))), and it loads three
 * apps with one listEnvironments request each.
 *
 * Do not reach it through the nullable side of an outer join. TanStack DB does
 * not push that side's where clause into the load, so the projectId is lost.
 */
export const environments = createCollection<Environment, string>(
  queryCollectionOptions({
    queryClient,
    queryKey: (opts) => {
      const apps = appsInWhere(opts.where);
      return apps
        ? ["environments", ...apps.map((a) => `${a.projectId}:${a.appId}`).sort()]
        : ["environments"];
    },
    syncMode: "on-demand",
    retry: 3,
    queryFn: async (ctx) => {
      const apps = appsInWhere(ctx.meta?.loadSubsetOptions?.where);

      if (!apps) {
        throw new Error(
          "Query must include eq(collection.projectId, projectId) and an eq or inArray constraint on collection.appId, or use inProjectApps",
        );
      }

      const perApp = await Promise.all(
        apps.map(async ({ projectId, appId }) => {
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

export type ProjectApps = { id: string; apps: ReadonlyArray<{ id: string }> };

/** Builds the where clause that loads the environments of every app in the given projects. */
export function inProjectApps(
  env: Ref<Environment>,
  projects: ReadonlyArray<ProjectApps>,
): IR.BasicExpression<boolean> {
  const [first, second, ...rest] = projects.map((project) =>
    and(
      eq(env.projectId, project.id),
      inArray(
        env.appId,
        project.apps.map((app) => app.id),
      ),
    ),
  );
  if (!first) {
    throw new Error("inProjectApps needs at least one project");
  }
  return second ? or(first, second, ...rest) : first;
}

type AppScope = { projectId?: string; appIds?: string[] };

/**
 * Reads the apps a where clause asks for, as project and app pairs. Returns null
 * when a branch of the clause misses its project or its apps. Predicates on other
 * fields only narrow the rows, so they are left to the live query.
 *
 * @example
 * appsInWhere(
 *   or(
 *     and(eq(env.projectId, "proj_1"), inArray(env.appId, ["app_a", "app_b"])),
 *     and(eq(env.projectId, "proj_2"), eq(env.appId, "app_c")),
 *   ),
 * );
 * // [
 * //   { projectId: "proj_1", appId: "app_a" },
 * //   { projectId: "proj_1", appId: "app_b" },
 * //   { projectId: "proj_2", appId: "app_c" },
 * // ]
 *
 * appsInWhere(inArray(env.appId, ["app_a"])); // null, no projectId
 */
export function appsInWhere(
  where: IR.BasicExpression<boolean> | undefined,
): Array<{ projectId: string; appId: string }> | null {
  const scopes = parseWhereExpression<AppScope[]>(where, {
    handlers: {
      eq: (field: unknown, value: unknown) => scopeOf(field, [value]),
      in: (field: unknown, values: unknown) => scopeOf(field, Array.isArray(values) ? values : []),
      and: (...parts: AppScope[][]) =>
        parts.reduce((merged, part) =>
          merged.flatMap((left) =>
            part.map((right) => ({
              projectId: left.projectId ?? right.projectId,
              appIds:
                left.appIds && right.appIds
                  ? left.appIds.filter((id) => right.appIds?.includes(id))
                  : (left.appIds ?? right.appIds),
            })),
          ),
        ),
      or: (...parts: AppScope[][]) => parts.flat(),
    },
    onUnknownOperator: () => [{}],
  });

  if (!scopes || scopes.some((scope) => !scope.projectId || !scope.appIds?.length)) {
    return null;
  }

  const pairs = new Map<string, { projectId: string; appId: string }>();
  for (const { projectId, appIds } of scopes) {
    for (const appId of appIds ?? []) {
      if (projectId) {
        pairs.set(`${projectId}:${appId}`, { projectId, appId });
      }
    }
  }
  return [...pairs.values()];
}

function scopeOf(field: unknown, values: unknown[]): AppScope[] {
  const name = Array.isArray(field) ? field.at(-1) : undefined;
  const strings = values.filter((v): v is string => typeof v === "string");
  if (name === "projectId" && strings.length === 1) {
    return [{ projectId: strings[0] }];
  }
  if (name === "appId") {
    return [{ appIds: strings }];
  }
  return [{}];
}

/**
 * `environments.kind` as the database declares it. The table is imported as a
 * type only, so `satisfies` above breaks the build when the column drops a
 * value, and drizzle stays out of the browser bundle.
 */
type KindColumn = (typeof environmentsTable)["kind"]["enumValues"][number];
