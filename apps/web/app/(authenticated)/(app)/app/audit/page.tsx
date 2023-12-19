import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db, desc, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { DataTable } from "./table";
import { columns } from "./table-columns";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function AuditPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace?.betaFeatures.auditLogRetentionDays) {
    return notFound();
  }

  const since = new Date(
    Date.now() - workspace.betaFeatures.auditLogRetentionDays * 24 * 60 * 60 * 1000,
  );

  const logs = await db.query.auditLogs.findMany({
    where: (table, { eq, and, gte }) =>
      and(eq(table.workspaceId, workspace.id), gte(table.time, since)),
    orderBy: desc(schema.auditLogs.time),
    limit: 10_000,
  });

  return (
    <div>
      <PageHeader
        title="Audit Logs"
        description={`You have access to the last ${workspace.betaFeatures.auditLogRetentionDays} days.`}
      />

      <main className="mb-20 mt-8">
        <DataTable data={logs} columns={columns} />
      </main>
    </div>
  );
}
