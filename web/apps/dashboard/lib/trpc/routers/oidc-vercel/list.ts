import { and, db, eq, inArray, schema } from "@/lib/db";
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
      orderBy: (table, { desc }) => [desc(table.createdAt)],
    });

    const permissions =
      integrations.length > 0
        ? await db.query.unkeyPrincipalPermissions.findMany({
            where: and(
              eq(schema.unkeyPrincipalPermissions.workspaceId, ctx.workspace.id),
              eq(schema.unkeyPrincipalPermissions.principalType, "oidc"),
              inArray(
                schema.unkeyPrincipalPermissions.principalId,
                integrations.map(({ id }) => id),
              ),
            ),
            columns: { principalId: true, slug: true },
          })
        : [];

    return integrations.map((integration) => ({
      ...integration,
      permissions: permissions
        .filter(({ principalId }) => principalId === integration.id)
        .map(({ slug }) => ({ slug })),
    }));
  } catch (error) {
    console.error("Failed to list Vercel OIDC integrations", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "We couldn't load the Vercel OIDC integrations.",
    });
  }
});
