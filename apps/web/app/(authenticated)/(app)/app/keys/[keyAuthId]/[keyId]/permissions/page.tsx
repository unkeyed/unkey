import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";
import { CreateNewPermission } from "../../../../authorization/permissions/create-new-permission";
import { CreateNewRole } from "../../../../authorization/roles/create-new-role";
import { Chart } from "./chart";

type Props = {
  params: {
    keyId: string;
  };
};

export default async function (props: Props) {
  const key = await db.query.keys.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.id, props.params.keyId), isNull(table.deletedAt)),
    with: {
      roles: {
        with: {
          role: {
            with: {
              permissions: {
                with: {
                  permission: true,
                },
              },
            },
          },
        },
      },
      permissions: true,
      workspace: {
        with: {
          roles: {
            with: {
              permissions: true,
            },
          },
          permissions: {
            with: {
              roles: true,
            },
          },
        },
      },
    },
  });
  if (!key) {
    return notFound();
  }

  const transientPermissionIds = new Set<string>();
  const connectedRoleIds = new Set<string>();
  for (const role of key.roles) {
    connectedRoleIds.add(role.roleId);
  }
  for (const role of key.workspace.roles) {
    if (connectedRoleIds.has(role.id)) {
      for (const p of role.permissions) {
        transientPermissionIds.add(p.permissionId);
      }
    }
  }

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      <div className="flex items-center justify-between flex-1 w-full gap-2">
        <div className="flex items-center gap-2">
          <Badge variant="secondary" className="h-8">
            {Intl.NumberFormat().format(key.roles.length)} Roles{" "}
          </Badge>
          <Badge variant="secondary" className="h-8">
            {Intl.NumberFormat().format(transientPermissionIds.size)} Permissions
          </Badge>
        </div>
        <div className="flex items-center gap-2">
          <CreateNewRole trigger={<Button variant="secondary">Create New Role</Button>} />
          <CreateNewPermission
            trigger={<Button variant="secondary">Create New Permission</Button>}
          />
        </div>
      </div>

      <Chart
        key={JSON.stringify(key)}
        data={key}
        roles={key.workspace.roles.map((r) => ({
          ...r,
          active: key.roles.some((keyRole) => keyRole.roleId === r.id),
        }))}
        permissions={key.workspace.permissions.map((p) => ({
          ...p,
          active: transientPermissionIds.has(p.id),
        }))}
      />
    </div>
  );
}
