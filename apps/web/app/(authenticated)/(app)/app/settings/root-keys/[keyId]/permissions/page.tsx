import { getTenantId } from "@/lib/auth";
import { Permission, db, eq, schema } from "@/lib/db";
import { notFound } from "next/navigation";
import { Api } from "./api";
import { Legacy } from "./legacy";
import { Workspace } from "./workspace";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RootKeyPage(props: {
  params: { keyId: string };
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
    with: {
      permissions: {
        with: {
          permission: true,
        },
      },
    },
  });
  if (!key) {
    return notFound();
  }

  console.log(JSON.stringify({ key }, null, 2));

  const permissions = key.permissions.map((kp) => kp.permission);

  const permissionsByApi = permissions.reduce((acc, permission) => {
    if (!permission.name.startsWith("api.")) {
      return acc;
    }
    const [_, apiId, _action] = permission.name.split(".");

    if (!acc[apiId]) {
      acc[apiId] = [];
    }
    acc[apiId].push(permission);
    return acc;
  }, {} as { [apiId: string]: Permission[] });

  return (
    <div className="flex flex-col gap-4">
      {permissions.some((p) => p.name === "*") ? (
        <Legacy keyId={key.id} permissions={permissions} />
      ) : null}

      <Workspace keyId={key.id} permissions={permissions} />

      {workspace.apis.map((api) => (
        <Api key={api.id} api={api} keyId={key.id} permissions={permissionsByApi[api.id] || []} />
      ))}
    </div>
  );
}
