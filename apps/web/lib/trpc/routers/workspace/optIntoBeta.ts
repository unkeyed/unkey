import { db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const optWorkspaceIntoBeta = t.procedure
  .use(auth)
  .input(
    z.object({
      feature: z.enum(["rbac", "auditLogRetentionDays", "ratelimit"]),
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
        message: "workspace not found",
      });
    }

    switch (input.feature) {
      case "rbac": {
        workspace.betaFeatures.rbac = true;
        break;
      }
      case "auditLogRetentionDays": {
        workspace.betaFeatures.auditLogRetentionDays = 30;
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
      .where(eq(schema.workspaces.id, workspace.id));
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
