import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
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
  });
  if (!key) {
    return notFound();
  }

  const roles = await db.query.roles.findMany({
    where: (table, { eq }) => eq(table.keyId, props.params.keyId),
  });

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

  const permissions = roles.map((r) => r.role);
  console.log({ roles });
  return (
    <div className="flex flex-col gap-4">
      <Alert variant="warn">
        <AlertTitle>Preview</AlertTitle>
        <AlertDescription>
          While we are in beta, you can already assign permissions to your keys, but they are not
          yet enforced.
        </AlertDescription>
      </Alert>
      {permissions.some((p) => p === "*") ? (
        <Legacy keyId={key.id} permissions={permissions} />
      ) : null}

      <Workspace keyId={key.id} permissions={permissions} />

      {workspace.apis.map((api) => (
        <Api key={api.id} api={api} keyId={key.id} permissions={permissionsByApi[api.id] || []} />
      ))}
    </div>
  );
}
