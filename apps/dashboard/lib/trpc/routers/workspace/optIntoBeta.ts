import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const optWorkspaceIntoBeta = t.procedure
  .use(auth)
  .input(
    z.object({
      feature: z.enum(["rbac", "ratelimit", "identities", "logsPage"]),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable opt you in to this beta feature. Please try again or contact support@unkey.dev",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, Please try again or contact support@unkey.dev.",
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
      .transaction(async (tx) => {
        await tx
          .update(schema.workspaces)
          .set({
            betaFeatures: workspace.betaFeatures,
          })
          .where(eq(schema.workspaces.id, workspace.id));
        await insertAuditLogs(tx, {
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
      })
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to update workspace, Please try again or contact support@unkey.dev.",
        });
      });
  });
