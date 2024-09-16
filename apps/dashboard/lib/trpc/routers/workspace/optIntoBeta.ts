import { db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";


export const optWorkspaceIntoBeta = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      feature: z.enum(["rbac", "ratelimit", "identities"]),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    switch (input.feature) {
      case "rbac": {
        workspace.betaFeatures.rbac = true;
        break;
      }
      case "identities": {
        workspace.betaFeatures.identities = true;
        break;
      }
      case "ratelimit": {
        workspace.betaFeatures.ratelimit = true;
        break;
      }
    }
    await db
      .update(schema.workspaces)
      .set({
        betaFeatures: workspace.betaFeatures,
      })
      .where(eq(schema.workspaces.id, workspace.id))
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to update workspace, please contact support using support@unkey.dev.",
        });
      });
    await ingestAuditLogs({
      workspaceId: workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "workspace.opt_in",
      description: `Opted ${workspace.id} into beta: ${input.feature}`,
      resources: [
        {
          type: "workspace",
          id: workspace.id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
