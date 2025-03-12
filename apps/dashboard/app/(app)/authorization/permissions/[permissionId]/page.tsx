import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound, redirect } from "next/navigation";
import { Navigation } from "./navigation";
import { PermissionClient } from "./settings-client";

export const revalidate = 0;

type Props = {
  params: {
    permissionId: string;
  };
};

export default async function RolesPage(props: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
    with: {
      permissions: {
        where: (table, { eq }) => eq(table.id, props.params.permissionId),
        with: {
          keys: true,
          roles: {
            with: {
              role: {
                with: {
                  keys: {
                    columns: {
                      keyId: true,
                    },
                  },
                },
              },
            },
          },
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  const permission = workspace.permissions.at(0);

  if (!permission) {
    return notFound();
  }

  return (
    <div>
      <Navigation permissionId={props.params.permissionId} permission={permission} />
      <PermissionClient permission={permission} />
    </div>
  );
}
