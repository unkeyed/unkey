import { PageContent } from "@/components/page-content";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound, redirect } from "next/navigation";
import { Navigation } from "./navigation";
import { type NestedPermissions, Tree } from "./tree";

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

export default async function RolesPage(props: Props) {
  const tenantId = await getTenantId();

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

  const sortedPermissions = workspace.permissions.sort((a, b) => {
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
          checked: role.permissions.some((p) => p.permissionId === permission.id),
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
      <PageContent>
        <div className="flex flex-col min-h-screen gap-8">
          <div className="flex items-center justify-between">
            <div className="grow min-w-2 ml-1">
              <div className="flex items-center gap-2">
                <h2 className="text-2xl font-semibold tracking-tight truncate">{role.name}</h2>
              </div>
              <p className="text-xs text-content-subtle truncate">{role.description}</p>
            </div>
          </div>
          <Tree nestedPermissions={sortedNestedPermissions} role={{ id: role.id }} />
        </div>
      </PageContent>
    </div>
  );
}
