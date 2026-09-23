import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

export const getWorkspace = async (orgId: string, role: string) => {
  try {
    const [members, workspace] = await Promise.all([
      auth
        .getOrganizationMemberList(orgId)
        .then((membersOfOrg) =>
          membersOfOrg.data.map((m) => ({
            label: m.user.fullName ?? m.user.email,
            value: m.user.id,
          })),
        )
        .catch((memberError: unknown) => {
          console.error(
            `Failed to fetch organization members for tenant ID ${orgId}: ${
              memberError instanceof Error ? memberError.message : "Unknown error"
            }`,
          );
          return null;
        }),
      db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
        columns: { id: true },
      }),
    ]);

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

    return { workspace: { ...workspace, keys: rootKeys }, members };
  } catch (error) {
    console.error(
      `Failed to fetch workspace for tenant ID ${orgId}: ${
        error instanceof Error ? error.message : "Unknown error"
      }`,
    );
    throw error;
  }
};
