import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";

export const revalidate = 0;

export default async function DebugWorkspacePage() {
  const tenantId = getTenantId();

  const workspace = await db().query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return <div>Workspace with tenantId: {tenantId} not found</div>;
  }

  return <pre>{JSON.stringify(workspace, null, 2)}</pre>;
}
