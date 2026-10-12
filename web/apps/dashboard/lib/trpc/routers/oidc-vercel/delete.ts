import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const deleteOIDCVercelIntegration = workspaceProcedure
  .input(z.object({ id: z.string().min(1) }))
  .mutation(async ({ ctx, input }) => {
    try {
      await db.transaction(async (tx) => {
        const integration = await tx.query.oidcVercel.findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.id, input.id), eq(table.workspaceId, ctx.workspace.id)),
          columns: { id: true },
        });
        if (!integration) {
          throw new TRPCError({ code: "NOT_FOUND", message: "Integration not found." });
        }

        await tx
          .delete(schema.unkeyPrincipalPermissions)
          .where(
            and(
              eq(schema.unkeyPrincipalPermissions.principalId, input.id),
              eq(schema.unkeyPrincipalPermissions.principalType, "oidc"),
              eq(schema.unkeyPrincipalPermissions.workspaceId, ctx.workspace.id),
            ),
          );
        await tx
          .delete(schema.oidcVercel)
          .where(
            and(
              eq(schema.oidcVercel.id, input.id),
              eq(schema.oidcVercel.workspaceId, ctx.workspace.id),
            ),
          );
      });
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to delete Vercel OIDC integration", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We couldn't delete the Vercel OIDC integration.",
      });
    }
  });
