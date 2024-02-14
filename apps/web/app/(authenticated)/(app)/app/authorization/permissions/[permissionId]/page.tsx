import { CopyButton } from "@/components/dashboard/copy-button";
import { PageHeader } from "@/components/dashboard/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Activity, CalendarPlus, KeySquare, LucideIcon, Minus, SquareStack } from "lucide-react";
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
    <div className="flex flex-col min-h-screen gap-4">
      <PageHeader
        title={<span className="font-mono">{permission.name}</span>}
        description={permission.description ?? undefined}
        actions={[
          <Badge
            key="permission-name"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {permission.name}
            <CopyButton value={permission.name} />
          </Badge>,
          <Badge
            key="permission-id"
            variant="secondary"
            className="flex justify-between w-full gap-2 font-mono font-medium ph-no-capture"
          >
            {permission.id}
            <CopyButton value={permission.id} />
          </Badge>,
          <DeletePermission
            key="delete-permission"
            trigger={<Button variant="alert">Delete</Button>}
            permission={permission}
          />,
        ]}
      />

      <div className="flex gap-4 mt-8 mb-20">
        <div className="grid w-full grid-cols-1 gap-4 min-w-20 ">
          <Metric
            Icon={CalendarPlus}
            label="Created At"
            value={permission.createdAt?.toDateString()}
          />
          <Metric Icon={Activity} label="Updated At" value={permission.updatedAt?.toDateString()} />
          <Metric
            Icon={SquareStack}
            label="Connected Roles"
            value={permission.roles.length.toString()}
          />
          <Metric Icon={KeySquare} label="Connected Keys" value={connectedKeys.size.toString()} />
        </div>
        <Client permission={permission} />
      </div>
    </div>
  );
}

const Metric: React.FC<{ label: string; value?: string; Icon: LucideIcon }> = ({
  label,
  value,
  Icon,
}) => {
  return (
    <div className="flex items-center gap-4 px-4 py-2 border rounded-lg">
      <Icon className="w-6 h-6 text-primary" />
      <div className="flex flex-col items-start justify-center">
        <p className="text-sm text-content-subtle">{label}</p>
        <div className="text-2xl font-semibold leading-none tracking-tight">
          {value ?? <Minus className="w-4 h-4" />}
        </div>
      </div>
    </div>
  );
};
