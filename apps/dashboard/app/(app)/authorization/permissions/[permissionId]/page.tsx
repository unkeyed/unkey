import { PageContent } from "@/components/page-content";
import { Metric } from "@/components/ui/metric";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { format } from "date-fns";
import { notFound, redirect } from "next/navigation";
import { Client } from "./client";
import { Navigation } from "./navigation";

export const revalidate = 0;

type Props = {
  params: {
    permissionId: string;
  };
};

export default async function RolesPage(props: Props) {
  const tenantId = await getTenantId();

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

  const connectedKeys = new Set<string>();
  for (const key of permission.keys) {
    connectedKeys.add(key.keyId);
  }
  for (const role of permission.roles) {
    for (const key of role.role.keys) {
      connectedKeys.add(key.keyId);
    }
  }

  return (
    <div>
      <Navigation permissionId={props.params.permissionId} permission={permission} />
      <PageContent>
        <div className="flex flex-col min-h-screen gap-4">
          <div className="flex gap-4 mb-20">
            <div className="grid w-full grid-cols-1 gap-4 min-w-20 ">
              <Metric
                className="border rounded-lg"
                label="Created At"
                value={format(permission.createdAtM, "PPPP")}
              />
              <Metric
                className="border rounded-lg"
                label="Updated At"
                value={
                  permission.updatedAtM
                    ? format(new Date(permission.updatedAtM).toDateString(), "PPPP")
                    : "Not updated yet"
                }
              />
              <Metric
                className="border rounded-lg"
                label="Connected Roles"
                value={permission.roles.length.toString()}
              />
              <Metric
                className="border rounded-lg"
                label="Connected Keys"
                value={connectedKeys.size.toString()}
              />
            </div>
            <Client permission={permission} />
          </div>
        </div>
      </PageContent>
    </div>
  );
}
