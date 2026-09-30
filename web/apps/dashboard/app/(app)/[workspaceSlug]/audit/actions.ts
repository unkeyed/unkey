import { db } from "@/lib/db";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

export const getWorkspace = async (orgId: string, role: string) => {
  try {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
      columns: { id: true },
    });

    if (!workspace) {
      return redirect(routes.auth.signIn());
    }

    const rootKeys =
      role === "admin"
        ? await db.query.keys.findMany({
            where: (table, { eq }) => eq(table.forWorkspaceId, workspace.id),
            columns: {
              id: true,
              name: true,
            },
          })
        : [];

    return { workspace: { ...workspace, keys: rootKeys } };
  } catch (error) {
    console.error(
      `Failed to fetch workspace for tenant ID ${orgId}: ${
        error instanceof Error ? error.message : "Unknown error"
      }`,
    );
    throw error;
  }
};
