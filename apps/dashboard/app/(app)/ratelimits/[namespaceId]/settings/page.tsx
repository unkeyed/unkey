import { CopyButton } from "@/components/dashboard/copy-button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound, redirect } from "next/navigation";
import { DeleteNamespace } from "./delete-namespace";
import { UpdateNamespaceName } from "./update-namespace-name";

export const dynamic = "force-dynamic";

type Props = {
  params: {
    namespaceId: string;
  };
};

export default async function SettingsPage(props: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      ratelimitNamespaces: {
        where: (table, { eq }) => eq(table.id, props.params.namespaceId),
      },
    },
  });
  if (!workspace || workspace.tenantId !== tenantId) {
    return redirect("/new");
  }
  const namespace = workspace.ratelimitNamespaces.find(
    (namespace) => namespace.id === props.params.namespaceId,
  );
  if (!namespace) {
    return notFound();
  }

  return (
    <div className="flex flex-col gap-8 mb-20 ">
      <UpdateNamespaceName namespace={namespace} />
      <Card>
        <CardHeader>
          <CardTitle>Namespace ID</CardTitle>
          <CardDescription>This is your namespace id. It's used in some API calls.</CardDescription>
        </CardHeader>
        <CardContent>
          <Code className="flex items-center justify-between w-full h-8 max-w-sm gap-4">
            <pre>{namespace.id}</pre>
            <div className="flex items-start justify-between gap-4">
              <CopyButton value={namespace.id} />
            </div>
          </Code>
        </CardContent>
      </Card>
      <DeleteNamespace namespace={namespace} />
    </div>
  );
}
