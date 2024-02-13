import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { Permission, db } from "@/lib/db";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@radix-ui/react-tooltip";
import { ChevronRight } from "lucide-react";
import { notFound, redirect } from "next/navigation";
import { DeleteRole } from "./delete-role";
import { PermissionToggle } from "./permission-toggle";
import { UpdateRole } from "./update-role";

export const revalidate = 0;

type Props = {
  params: {
    roleId: string;
  };
};

type NestedPermission = {
  id: string;
  name: string;
  checked: boolean;
  part: string;
  path: string;
  permissions: NestedPermissions;
  level: number;
};

type NestedPermissions = Record<string, NestedPermission>;

export default async function RolesPage(props: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
    with: {
      roles: {
        where: (table, { eq }) => eq(table.id, props.params.roleId),
        with: {
          permissions: true,
        },
      },
      permissions: {
        with: {
          roles: true,
        },
        orderBy: (table, { asc }) => [asc(table.name)],
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

  const nested: NestedPermissions = {};
  for (const permission of workspace.permissions) {
    let n = nested;
    const parts = permission.name.split(".");
    for (let i = 0; i < parts.length; i++) {
      const p = parts[i];
      if (!(p in n)) {
        n[p] = {
          id: permission.id,
          name: permission.name,
          checked: role.permissions.some((p) => p.permissionId === permission.id),
          part: p,
          permissions: {},
          level: i,
          path: parts.slice(0, i).join("."),
        };
      }
      n = n[p].permissions;
    }
  }

  return (
    <div className="flex flex-col min-h-screen gap-8">
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="font-mono text-2xl font-semibold tracking-tight">{role.name}</h2>
          </div>
          <p className="text-xs text-content-subtle">{role.description}</p>
        </div>
        <div className="flex items-center gap-2">
          <UpdateRole role={role} trigger={<Button variant="secondary">Update Role</Button>} />
          <DeleteRole role={role} trigger={<Button variant="alert">Delete Role</Button>} />
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Permissions</CardTitle>
          <CardDescription>Add or remove permissions from this role</CardDescription>
        </CardHeader>

        <CardContent className="flex flex-col gap-1 ">
          {Object.entries(nested).map(([k, p]) => (
            <RecursivePermission
              k={k}
              {...p}
              allPermissions={workspace.permissions}
              roleId={role.id}
            />
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

const RecursivePermission: React.FC<
  NestedPermission & { allPermissions: Permission[]; k: string; roleId: string }
> = ({ k, id, level, name, permissions, roleId, allPermissions, checked }) => {
  const permission = allPermissions.find((p) => p.id === id)!;

  const children = Object.values(permissions);

  if (children.length === 0) {
    return (
      <div className="px-2 py-1 ml-4 hover:bg-secondary">
        <p className="text-xs text-content-subtle">{permission.description}</p>
        <div className="flex items-center gap-2">
          <PermissionToggle permissionId={id} roleId={roleId} checked={checked} />
          {/* TODO  */}
          <pre className="text-sm">{k}</pre>
        </div>
      </div>
    );
  }
  return (
    <div className="p-2 ml-4">
      <div className="flex items-center gap-1">
        <ChevronRight className="w-4 h-4" />
        <pre className="text-sm">{k}</pre>
      </div>
      <div className="flex flex-col gap-1 ml-2 border-l border-border">
        {Object.entries(permissions).map(([k2, p]) => (
          <RecursivePermission
            key={p.id}
            k={k2}
            {...p}
            allPermissions={allPermissions}
            roleId={roleId}
          />
        ))}
      </div>
    </div>
  );
};
