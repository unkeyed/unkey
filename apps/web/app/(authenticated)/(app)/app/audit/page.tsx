import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db, desc, schema } from "@/lib/db";
import { clerkClient } from "@clerk/nextjs";
import { User } from "@clerk/nextjs/server";
import { notFound } from "next/navigation";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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

      <main className="mt-8 mb-20">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {/* {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center">
                  No results.
                </TableCell>
              </TableRow>
            )} */}
          </TableBody>
        </Table>
      </main>
    </div>
  );
}
