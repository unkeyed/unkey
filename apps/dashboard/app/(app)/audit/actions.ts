import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";

export const getWorkspace = async (orgId: string) => {
  try {
    let members = null;

    try {
      const membersOfOrg = await auth.getOrganizationMemberList(orgId);
      members = membersOfOrg.data.map((m) => ({
        label: m.user.fullName ?? m.user.email,
        value: m.user.id,
      }));
    } catch (memberError) {
      console.error(
        `Failed to fetch organization members for tenant ID ${orgId}: ${
          memberError instanceof Error ? memberError.message : "Unknown error"
        }`,
      );
    }

    const workspace = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    });

    if (!workspace) {
      return redirect("/auth/sign-in");
    }

    const rootKeys = await db.query.keys.findMany({
      where: (table, { eq }) => eq(table.forWorkspaceId, workspace?.id),

      columns: {
        id: true,
        name: true,
      },
    });

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
