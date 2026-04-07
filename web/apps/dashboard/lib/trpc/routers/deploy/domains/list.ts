import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const listDomains = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    const project = await db.query.projects
      .findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          id: true,
        },
        with: {
          frontlineRoutes: {
            columns: {
              id: true,
              projectId: true,
              deploymentId: true,
              environmentId: true,
              fullyQualifiedDomainName: true,
              sticky: true,
              createdAt: true,
              updatedAt: true,
            },
            where: (table, { eq }) => eq(table.routeType, "deployment"),
            limit: 500,
            orderBy: (table, { desc }) => desc(table.updatedAt),
          },
        },
      })
      .catch((error) => {
        console.error("Error querying domains:", error);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve domains due to an error. If this issue persists, please contact support.",
        });
      });

    // Filter to deployment routes with non-null deploy fields.
    // The `where` clause above ensures routeType='deployment', but Drizzle's
    // inferred type still includes nullable fields from the schema. We narrow
    // at runtime so the return type satisfies the Domain collection schema.
    return (project?.frontlineRoutes ?? []).flatMap((r) => {
      if (r.projectId == null || r.deploymentId == null || r.environmentId == null) {
        return [];
      }
      return [
        {
          ...r,
          projectId: r.projectId,
          deploymentId: r.deploymentId,
          environmentId: r.environmentId,
        },
      ];
    });
  });
