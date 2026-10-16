import { and, db, eq, inArray, schema } from "@/lib/db";
import { githubAppEnv } from "@/lib/env";
import { installationExists } from "@/lib/github";

// ctrl deletes bindings when GitHub sends installation.deleted. Bindings from
// before that handler, or from a missed event, stay until GitHub says the
// installation is gone, so every "is GitHub connected?" read reconciles first.
export async function reconcileInstallations(workspaceId: string): Promise<number[]> {
  const bindings = await db.query.githubAppInstallations.findMany({
    where: (table, { eq }) => eq(table.workspaceId, workspaceId),
    columns: { installationId: true },
  });
  if (bindings.length === 0 || !githubAppEnv()) {
    return bindings.map((b) => b.installationId);
  }

  const checked = await Promise.all(
    bindings.map(async ({ installationId }) => ({
      installationId,
      exists: await installationExists(installationId).catch((err: unknown) => {
        console.error(err);
        return true;
      }),
    })),
  );
  const gone = checked.filter((c) => !c.exists).map((c) => c.installationId);
  if (gone.length > 0) {
    await db.transaction(async (tx) => {
      await tx
        .delete(schema.githubRepoConnections)
        .where(
          and(
            eq(schema.githubRepoConnections.workspaceId, workspaceId),
            inArray(schema.githubRepoConnections.installationId, gone),
          ),
        );
      await tx
        .delete(schema.githubAppInstallations)
        .where(
          and(
            eq(schema.githubAppInstallations.workspaceId, workspaceId),
            inArray(schema.githubAppInstallations.installationId, gone),
          ),
        );
    });
  }
  return checked.filter((c) => c.exists).map((c) => c.installationId);
}
