import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound, redirect } from "next/navigation";
import { Navigation } from "./navigation";
import { RoleClient } from "./settings-client";
import type { NestedPermissions } from "./settings-client";

export const revalidate = 0;

type Props = {
  params: {
    roleId: string;
  };
};

function sortNestedPermissions(nested: NestedPermissions) {
  const shallowPermissions: NestedPermissions = {};
  const nestedPermissions: NestedPermissions = {};

  for (const [key, value] of Object.entries(nested)) {
    if (Object.keys(value.permissions).length > 0) {
      nestedPermissions[key] = value;
    } else {
      shallowPermissions[key] = value;
    }
  }

  const sortedShallowKeys = Object.keys(shallowPermissions).sort();
  const sortedNestedKeys = Object.keys(nestedPermissions).sort();

  const sortedObject: NestedPermissions = {};

  for (const key of sortedShallowKeys) {
    sortedObject[key] = shallowPermissions[key];
  }

  for (const key of sortedNestedKeys) {
    sortedObject[key] = nestedPermissions[key];
  }

  return sortedObject;
}

export default async function RolePage(props: Props) {
  const tenantId = getTenantId();

  // Get workspace with active permissions only
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq, isNull }) => eq(table.tenantId, tenantId) && isNull(table.deletedAtM),
    with: {
      roles: {
        where: (table, { eq }) => eq(table.id, props.params.roleId),
        with: {
          permissions: true,
          keys: {
            with: {
              key: {
                columns: {
                  deletedAtM: true,
                },
              },
            },
          },
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

  // Filter out soft-deleted keys
  const activeKeys = role.keys?.filter(({ key }) => key.deletedAtM === null) || [];

  // Get all active permissions in the workspace
  const activePermissions = workspace.permissions;

  // Filter role's permissions to only include active ones
  const activeRolePermissionIds = new Set(
    role.permissions
      .filter((rp) => activePermissions.some((p) => p.id === rp.permissionId))
      .map((rp) => rp.permissionId),
  );

  const sortedPermissions = activePermissions.sort((a, b) => {
    const aParts = a.name.split(".");
    const bParts = b.name.split(".");
    if (aParts.length !== bParts.length) {
      return aParts.length - bParts.length;
    }
    return a.name.localeCompare(b.name);
  });

  const nested: NestedPermissions = {};

  for (const permission of sortedPermissions) {
    let n = nested;
    const parts = permission.name.split(".");

    for (let i = 0; i < parts.length; i++) {
      const p = parts[i];

      if (!(p in n)) {
        n[p] = {
          id: permission.id,
          name: permission.name,
          description: permission.description,
          checked: activeRolePermissionIds.has(permission.id),
          part: p,
          permissions: {},
          path: parts.slice(0, i).join("."),
        };
      }

      n = n[p].permissions;
    }
  }

  const sortedNestedPermissions = sortNestedPermissions(nested);

  return (
    <div>
      <Navigation role={role} />
      <RoleClient
        role={{
          ...role,
          permissions: role.permissions.filter((p) => activeRolePermissionIds.has(p.permissionId)),
        }}
        activeKeys={activeKeys}
        sortedNestedPermissions={sortedNestedPermissions}
      />
    </div>
  );
}
