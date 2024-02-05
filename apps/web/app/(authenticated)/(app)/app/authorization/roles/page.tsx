import { PageHeader } from "@/components/dashboard/page-header";
import { RootKeyTable } from "@/components/dashboard/root-key-table";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { CreateNewRole } from "./create-new-role";
import { DataTable } from "./table";
import { columns } from "./table-columns";

export const revalidate = 0;

export default async function RolesPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      roles: {
        with: {
          permissions: {
            with: {
              permission: true,
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
      <div className="grid w-full grid-cols-1 gap-8 mb-20">
        <DataTable
          data={workspace.roles.map((r) => ({
            ...r,
            permissions: r.permissions.map((p) => p.permission.name),
          }))}
          columns={columns}
        />
      </div>
    </div>
  );
}
