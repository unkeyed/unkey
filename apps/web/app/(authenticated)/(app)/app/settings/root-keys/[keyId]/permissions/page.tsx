import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { apiActions } from "@unkey/rbac";
import { notFound } from "next/navigation";
import { Api } from "./api";
import { Permission } from "./permission";
import { Workspace } from "./workspace";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RootKeyPage(props: {
  params: { keyId: string };
  searchParams: {
    interval?: Interval;
  };
}) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });
  if (!workspace) {
    return notFound();
  }

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.forWorkspaceId, workspace.id) && eq(schema.keys.id, props.params.keyId),
  });
  if (!key) {
    return notFound();
  }

  const roles = await db.query.roles.findMany({
    where: (table, { eq }) => eq(table.keyId, props.params.keyId),
  });

  console.log({ key, roles });

  const permissionsByApi = roles.reduce((acc, { role }) => {
    if (!role.startsWith("api.")) {
      return acc;
    }
    const [_, apiId, permission] = role.split(".");

    if (!acc[apiId]) {
      acc[apiId] = [];
    }
    acc[apiId].push(permission);
    return acc;
  }, {} as { [apiId: string]: string[] });

  return (
    <div className="flex flex-col gap-4">
      <Workspace keyId={key.id} permissions={roles.map((r) => r.role)} />

      {workspace.apis.map((api) => (
        <Api key={api.id} api={api} keyId={key.id} permissions={permissionsByApi[api.id] || []} />
      ))}
    </div>
  );
}
