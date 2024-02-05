import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound, redirect } from "next/navigation";
import { DataTable } from "./table";
import { columns } from "./table-columns";

export const revalidate = 0;

type Props = {
  params: {
    roleId: string;
  };
};

export default async function RolesPage(props: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
    with: {
      roles: {
        where: (table, { eq }) => eq(table.id, props.params.roleId),
      },
      permissions: {
        with: {
          roles: true,
        },
      },
    },
  });
  if (!workspace) {
    return redirect("/new");
  }

  const role = workspace.roles.at(0);
  if (!role) {
    return notFound();
  }
  return (
    <div className="min-h-screen ">
      <PageHeader title={role.name} description={role.description ?? undefined} />
      <Badge variant="secondary">{role.key}</Badge>
      <div className="mt-8 mb-20">
        <DataTable
          data={workspace.permissions.map((p) => ({
            id: p.id,
            name: p.name,
            roleId: role.id,
            checked: p.roles.some((r) => r.roleId === role.id),
          }))}
          columns={columns}
        />
      </div>
    </div>
  );
}
