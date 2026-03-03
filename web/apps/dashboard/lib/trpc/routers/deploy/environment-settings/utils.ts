import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { apps, environments } from "@unkey/db/src/schema";

/**
 * Resolves all app IDs for the project that the given environment belongs to.
 * Used to fan out settings changes to all apps in a project.
 */
export async function resolveProjectAppIds(
  workspaceId: string,
  environmentId: string,
): Promise<string[]> {
  const env = await db.query.environments.findFirst({
    where: and(eq(environments.id, environmentId), eq(environments.workspaceId, workspaceId)),
    columns: {},
    with: {
      project: {
        columns: {},
        with: {
          apps: {
            columns: { id: true },
          },
        },
      },
    },
  });

  if (!env) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Environment not found",
    });
  }

  return env.project.apps.map((a) => a.id);
}
