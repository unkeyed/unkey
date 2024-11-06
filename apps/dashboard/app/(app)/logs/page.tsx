import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";

export const revalidate = 0;

export default async function Page() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return <div>Workspace with tenantId: {tenantId} not found</div>;
  }

  const logs = await clickhouse.api.logs({ workspaceId: workspace.id, limit: 10 });

  return (
    <div>
      <PageHeader title="Logs" />

      <pre>{JSON.stringify(logs, null, 2)}</pre>
    </div>
  );
}
