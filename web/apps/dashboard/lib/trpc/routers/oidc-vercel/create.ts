import { db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { workspaceProcedure } from "../../trpc";
import { assertPermissionsBelongToWorkspace, upsertPermissions } from "../rbac";
import { oidcVercelInputSchema } from "./validation";

export const createOIDCVercelIntegration = workspaceProcedure
  .input(oidcVercelInputSchema)
  .mutation(async ({ ctx, input }) => {
    assertPermissionsBelongToWorkspace(input.permissions, ctx.workspace.id);
    const id = newId("oidcVercel");
    const audience = `https://api.unkey.com/oidc/vercel/${id}`;

    try {
      await db.transaction(async (tx) => {
        const permissions =
          input.permissions.length > 0
            ? (await upsertPermissions(tx, ctx, env().UNKEY_WORKSPACE_ID, input.permissions))
                .permissions
            : [];

        await tx.insert(schema.oidcVercel).values({
          id,
          workspaceId: ctx.workspace.id,
          name: input.name,
          audience,
          vercelTeamSlug: input.vercelTeamSlug,
          vercelProjectName: input.vercelProjectName,
          vercelEnvironment: input.vercelEnvironment,
          enabled: true,
        });

        if (permissions.length > 0) {
          await tx.insert(schema.oidcVercelPermissions).values(
            permissions.map((permission) => ({
              oidcVercelId: id,
              permissionId: permission.id,
              workspaceId: ctx.workspace.id,
            })),
          );
        }
      });
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to create Vercel OIDC integration", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We couldn't create the Vercel OIDC integration.",
      });
    }

    return {
      id,
      audience,
    };
  });
