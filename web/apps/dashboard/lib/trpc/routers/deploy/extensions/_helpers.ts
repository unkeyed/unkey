import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";

/**
 * Asserts the project belongs to the workspace, throwing NOT_FOUND otherwise.
 *
 * Used by every extension procedure so the workspace boundary check stays
 * uniform — no procedure forgets it, none implements it slightly differently.
 */
export async function assertProjectInWorkspace(projectId: string, workspaceId: string) {
  const project = await db.query.projects.findFirst({
    where: (table, { eq, and }) => and(eq(table.id, projectId), eq(table.workspaceId, workspaceId)),
    columns: { id: true },
  });
  if (!project) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Project not found" });
  }
}

/**
 * Loads an installation owned by the workspace. NOT_FOUND when the row is
 * missing, soft-deleted, or belongs to a different workspace — never leak the
 * existence of installations across workspaces.
 */
export async function getOwnedInstallation(installationId: string, workspaceId: string) {
  const row = await db.query.extensionInstallations.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(
        eq(table.id, installationId),
        eq(table.workspaceId, workspaceId),
        isNull(table.deletedAt),
      ),
  });
  if (!row) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Installation not found" });
  }
  return row;
}
