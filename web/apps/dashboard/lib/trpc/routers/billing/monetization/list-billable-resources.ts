import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

/**
 * Lists the workspace's keyspaces (APIs) and ratelimit namespaces with a flag
 * for whether each is enabled for end-user billing. A keyspace's billing id is
 * its key_auth id (the ClickHouse key_space_id), surfaced under the API's name;
 * a namespace's id is its own. Enablement is opt-in: only resources with a row
 * in billing_billable_resources are billed.
 */
export const listBillableResources = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    const [apis, namespaces, enabled] = await Promise.all([
      db.query.apis.findMany({
        where: (table, { and, eq, isNull, isNotNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
            isNotNull(table.keyAuthId),
          ),
        orderBy: (table, { asc }) => [asc(table.name)],
      }),
      db.query.ratelimitNamespaces.findMany({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.workspaceId, ctx.workspace.id), isNull(table.deletedAtM)),
        orderBy: (table, { asc }) => [asc(table.name)],
      }),
      db.query.billingBillableResources.findMany({
        where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
      }),
    ]);

    const enabledKeyspaces = new Set(
      enabled.filter((r) => r.resourceType === "keyspace").map((r) => r.resourceId),
    );
    const enabledNamespaces = new Set(
      enabled.filter((r) => r.resourceType === "namespace").map((r) => r.resourceId),
    );

    return {
      keyspaces: apis.flatMap((api) =>
        // keyAuthId is non-null by the query filter; flatMap narrows it here
        // without an unsafe cast.
        api.keyAuthId
          ? [
              {
                resourceId: api.keyAuthId,
                apiId: api.id,
                name: api.name,
                enabled: enabledKeyspaces.has(api.keyAuthId),
              },
            ]
          : [],
      ),
      namespaces: namespaces.map((ns) => ({
        resourceId: ns.id,
        name: ns.name,
        enabled: enabledNamespaces.has(ns.id),
      })),
    };
  });
