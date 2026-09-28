import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";

export const listOIDCVercelIntegrations = workspaceProcedure.query(async ({ ctx }) => {
  try {
    const integrations = await db.query.oidcVercel.findMany({
      where: eq(schema.oidcVercel.workspaceId, ctx.workspace.id),
      columns: {
        id: true,
        name: true,
        audience: true,
        vercelTeamSlug: true,
        vercelProjectName: true,
        vercelEnvironment: true,
        enabled: true,
      },
      with: {
        permissions: {
          columns: { permissionId: true },
          with: {
            permission: {
              columns: {
                id: true,
                name: true,
                slug: true,
                description: true,
              },
            },
          },
        },
      },
      orderBy: (table, { desc }) => [desc(table.createdAt)],
    });

    return integrations.map(({ permissions, ...integration }) => ({
      ...integration,
      permissions: permissions.map(({ permission }) => permission),
    }));
  } catch (error) {
    console.error("Failed to list Vercel OIDC integrations", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "We couldn't load the Vercel OIDC integrations.",
    });
  }
});
