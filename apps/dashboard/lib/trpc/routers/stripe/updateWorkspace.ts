import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { clearWorkspaceCache } from "../workspace/getCurrent";

export const updateWorkspaceStripeCustomer = workspaceProcedure
  .input(
    z.object({
      stripeCustomerId: z.string().min(1, "Stripe customer ID is required"),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.workspaces)
          .set({
            stripeCustomerId: input.stripeCustomerId,
          })
          .where(eq(schema.workspaces.id, ctx.workspace.id));

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: "Updated Stripe customer ID",
          resources: [
            {
              type: "workspace",
              id: ctx.workspace.id,
              name: ctx.workspace.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the workspace Stripe customer. Please try again or contact support@unkey.dev",
        });
      });
    // Invalidate workspace cache after successful update
    await invalidateWorkspaceCache(ctx.tenant.id);

    // Also clear the tRPC workspace cache to ensure fresh data on next request
    clearWorkspaceCache(ctx.tenant.id);
    return {
      success: true,
    };
  });
