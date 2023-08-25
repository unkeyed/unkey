import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema, type Key } from "@/lib/db";
import { redirect } from "next/navigation";
import { ApiKeyTable } from "@/components/dashboard/api-key-table";
import { CreateRootKeyButton } from "./create-root-key-button";

export const revalidate = 0;

export default async function SettingsKeysPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: {
        limit: 1,
      },
    },
  });
  if (!workspace) {
    return redirect("/onboarding");
  }

  const found = await db.query.keys.findMany({
    where: eq(schema.keys.forWorkspaceId, workspace.id),
    limit: 100,
  });

  const keys: Key[] = [];
  const expired: Key[] = [];

  for (const k of found) {
    if (k.expires && k.expires.getTime() < Date.now()) {
      expired.push(k);
    }
    // remove temp keys from the list of keys.
    if (!k.expires) {
      keys.push(k);
    }
  }
  if (expired.length > 0) {
    await Promise.all(expired.map((k) => db.delete(schema.keys).where(eq(schema.keys.id, k.id))));
  }

  return (
    <div>
      <PageHeader
        title="Keys"
        description="These keys are used to interact with the unkey API"
        actions={[<CreateRootKeyButton key="create-key" apiId={workspace.apis.at(0)?.id} />]}
      />
      <Separator className="my-6" />

      <ApiKeyTable data={keys} apiId={props.params.apiId} />
    </div>
  );
}
