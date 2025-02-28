import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Badge } from "@/components/ui/badge";
import { Metric } from "@/components/ui/metric";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { ShieldKey } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { format } from "date-fns";
import { notFound, redirect } from "next/navigation";
import { Client } from "./client";
import { DeletePermission } from "./delete-permission";

export const revalidate = 0;

type Props = {
  params: {
    permissionId: string;
  };
};

export default async function RolesPage(props: Props) {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.orgId, orgId),
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
  const shouldShowTooltip = permission.name.length > 16;

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<ShieldKey />}>
          <Navbar.Breadcrumbs.Link href="/authorization/roles">
            Authorization
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="/authorization/permissions">
            Permissions
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={`/authorization/permissions/${props.params.permissionId}`}
            isIdentifier
            active
          >
            {props.params.permissionId}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Badge
            key="permission-name"
            variant="secondary"
            className="w-40 font-mono font-medium ph-no-capture"
          >
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="flex items-center justify-between gap-2 w-full truncate">
                  <span className="truncate">{permission.name}</span>
                  <div>
                    <CopyButton value={permission.name} />
                  </div>
                </div>
              </TooltipTrigger>
              {shouldShowTooltip && (
                <TooltipContent>
                  <span className="text-xs font-medium">{permission.name}</span>
                </TooltipContent>
              )}
            </Tooltip>
          </Badge>
          <Badge
            key="permission-id"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {permission.id}
            <CopyButton value={permission.id} />
          </Badge>
          <DeletePermission
            key="delete-permission"
            trigger={<Button variant="destructive">Delete</Button>}
            permission={permission}
          />{" "}
        </Navbar.Actions>
      </Navbar>
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
