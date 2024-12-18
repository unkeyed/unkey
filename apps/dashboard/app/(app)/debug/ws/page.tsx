import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";

export const revalidate = 0;

export default async function DebugWorkspacePage() {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    return <div>Workspace with tenantId: {tenantId} not found</div>;
  }

  return <pre>{JSON.stringify(workspace, null, 2)}</pre>;
}
