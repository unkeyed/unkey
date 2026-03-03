import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";

/**
 * Resolves all environment IDs for the project that the given environment belongs to.
 * Used to fan out settings changes to all environments in a project.
 */
export async function resolveProjectEnvironmentIds(
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
          environments: {
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

  return env.project.environments.map((e) => e.id);
}
