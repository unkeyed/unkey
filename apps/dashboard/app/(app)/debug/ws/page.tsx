import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";

export const revalidate = 0;

export default async function DebugWorkspacePage() {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    return <div>Workspace with orgId: {orgId} not found</div>;
  }

  return <pre>{JSON.stringify(workspace, null, 2)}</pre>;
}
