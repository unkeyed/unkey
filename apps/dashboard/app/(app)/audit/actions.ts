import { auth } from "@/lib/auth/server";
import { type Workspace, db } from "@/lib/db";
import { redirect } from "next/navigation";

export async function getWorkspace(orgId: string): Promise<{
  workspace: Workspace & { auditLogBuckets: Array<{ id: string; name: string }> };
  rootKeys: Array<{ id: string; name: string | null }>;
  members: Array<{
    name: string;
    id: string;
  }>;
}> {
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

  const [rootKeys, members] = await Promise.all([
    db.query.keys.findMany({
      where: (table, { eq }) => eq(table.forWorkspaceId, workspace.id),

      columns: {
        id: true,
        name: true,
      },
    }),
    auth.getOrganizationMemberList(orgId),
  ]);

  return {
    workspace,
    rootKeys,
    members: members.data.map((m) => ({
      name: m.user.email,
      id: m.user.id,
    })),
  };
}
