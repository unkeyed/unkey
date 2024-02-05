import { PageHeader } from "@/components/dashboard/page-header";
import { RootKeyTable } from "@/components/dashboard/root-key-table";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { CreateNewRole } from "./create-new-role";
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
        where: (table, { eq }) => eq(table.publicId, props.params.roleId),
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
  const role = workspace.roles.at(0);
  if (!role) {
    return notFound();
  }

  return (
    <div className="min-h-screen ">
      <PageHeader
        title="Roles"
        description="Manage all roles in your workspace"
        actions={[
          <CreateNewRole
            key="create-new-role"
            trigger={<Button variant="primary">Create New Role</Button>}
          />,
        ]}
      />
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
