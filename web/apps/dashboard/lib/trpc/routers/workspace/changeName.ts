import { insertAuditLogs } from "@/lib/audit";
import { auth as authClient } from "@/lib/auth/server";
import { db, eq, schema } from "@/lib/db";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { clearWorkspaceCache } from "./getCurrent";

export const changeWorkspaceName = workspaceProcedure
  .input(
    z.object({
      name: z
        .string()
        .min(3, "Workspace names must contain at least 3 characters")
        .max(50, "Workspace names must contain less than 50 characters"),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    if (input.workspaceId !== ctx.workspace.id) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Invalid workspace ID",
      });
    }
    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.workspaces)
          .set({
            name: input.name,
          })
          .where(eq(schema.workspaces.id, input.workspaceId))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to update the workspace name. Please try again or contact support@unkey.dev",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: `Changed name from ${ctx.workspace.name} to ${input.name}`,
          resources: [
            {
              type: "workspace",
              id: ctx.workspace.id,
              name: input.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });

        await authClient.updateOrg({
          id: ctx.tenant.id,
          name: input.name,
        });

        // Clear both server-side and workspace caches after successful update
        clearWorkspaceCache(ctx.tenant.id);
        await invalidateWorkspaceCache(ctx.tenant.id);
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the workspace name. Please try again or contact support@unkey.dev",
        });
      });
  });
