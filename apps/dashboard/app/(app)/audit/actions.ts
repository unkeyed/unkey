import { db } from "@/lib/db";
import { clerkClient } from "@clerk/nextjs";
import { redirect } from "next/navigation";

export const getWorkspace = async (orgId: string) => {
  try {
    let members = null;

    if (!orgId.startsWith("user_")) {
      try {
        const membersOfOrg = await clerkClient.organizations.getOrganizationMembershipList({
          organizationId: orgId,
        });
        members = membersOfOrg
          .filter((m) => Boolean(m.publicUserData))
          .map((m) => ({
            label:
              m.publicUserData!.firstName && m.publicUserData!.lastName
                ? `${m.publicUserData!.firstName} ${m.publicUserData!.lastName}`
                : m.publicUserData!.identifier,
            value: m.publicUserData!.userId,
          }));
      } catch (memberError) {
        console.error(
          `Failed to fetch organization members for tenant ID ${orgId}: ${
            memberError instanceof Error ? memberError.message : "Unknown error"
          }`,
        );
      }
    }

    const workspace = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
      with: {
        auditLogBuckets: {
          columns: {
            id: true,
            name: true,
          },
          orderBy: (table, { asc }) => asc(table.createdAt),
        },
      },
    });

    if (!workspace) {
      return redirect("/auth/signin");
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
