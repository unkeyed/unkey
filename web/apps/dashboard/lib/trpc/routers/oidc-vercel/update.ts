import { and, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { assertPermissionsBelongToWorkspace, upsertPermissions } from "../rbac";
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

        const permissions =
          input.permissions.length > 0
            ? (await upsertPermissions(tx, ctx, env().UNKEY_WORKSPACE_ID, input.permissions))
                .permissions
            : [];

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
          .delete(schema.oidcVercelPermissions)
          .where(
            and(
              eq(schema.oidcVercelPermissions.oidcVercelId, input.id),
              eq(schema.oidcVercelPermissions.workspaceId, ctx.workspace.id),
            ),
          );
        if (permissions.length > 0) {
          await tx.insert(schema.oidcVercelPermissions).values(
            permissions.map((permission) => ({
              oidcVercelId: input.id,
              permissionId: permission.id,
              workspaceId: ctx.workspace.id,
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
