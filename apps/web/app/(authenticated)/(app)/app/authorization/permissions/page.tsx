import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { CreateNewPermission } from "./create-new-permission";
import { DataTable } from "./table";
import { columns } from "./table-columns";

export const revalidate = 0;

export default async function RolesPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      permissions: {
        with: {
          roles: {
            with: {
              role: true,
            },
          },
        },
      },
    },
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div className="min-h-screen ">
      <div className="mb-20">
        <DataTable
          data={workspace.permissions.map((p) => ({
            id: p.id,
            name: p.name,

            roles: p.roles.map((r) => r.role.key),
          }))}
          columns={columns}
        />
      </div>
    </div>
  );
}
