import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";

import { notFound } from "next/navigation";

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
    <div className="">
      <Card>
        <CardHeader>
          <CardTitle>Permissions</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-1">
          {Object.entries(permissionsByApi).map(([apiId, permissions]) => (
            <Card key={apiId}>
              <CardHeader>
                <CardTitle>{apiId}</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex flex-col gap-1">
                  {permissions.map((permission) => (
                    <div key={permission} className="flex items-center gap-1">
                      <Checkbox checked={true} />
                      <Label>{permission}</Label>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
