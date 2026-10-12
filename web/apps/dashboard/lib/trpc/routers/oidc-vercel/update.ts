import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { assertPermissionsBelongToWorkspace } from "../rbac";
import { oidcVercelInputSchema } from "./validation";

export const updateOIDCVercelIntegration = workspaceProcedure
  .input(
    oidcVercelInputSchema.extend({
      id: z.string().min(1),
      enabled: z.boolean(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    assertPermissionsBelongToWorkspace(input.permissions, ctx.workspace.id);
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
          .update(schema.oidcVercel)
          .set({
            name: input.name,
            vercelTeamSlug: input.vercelTeamSlug,
            vercelProjectName: input.vercelProjectName,
            vercelEnvironment: input.vercelEnvironment,
            enabled: input.enabled,
          })
          .where(
            and(
              eq(schema.oidcVercel.id, input.id),
              eq(schema.oidcVercel.workspaceId, ctx.workspace.id),
            ),
          );

        await tx
          .delete(schema.unkeyPrincipalPermissions)
          .where(
            and(
              eq(schema.unkeyPrincipalPermissions.principalId, input.id),
              eq(schema.unkeyPrincipalPermissions.principalType, "oidc"),
              eq(schema.unkeyPrincipalPermissions.workspaceId, ctx.workspace.id),
            ),
          );
        if (input.permissions.length > 0) {
          await tx.insert(schema.unkeyPrincipalPermissions).values(
            input.permissions.map((slug) => ({
              id: newId("permission"),
              principalId: input.id,
              principalType: "oidc" as const,
              workspaceId: ctx.workspace.id,
              slug,
              createdAt: Date.now(),
            })),
          );
        }
      });
      return { id: input.id };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to update Vercel OIDC integration", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We couldn't save the Vercel OIDC integration.",
      });
    }
  });
