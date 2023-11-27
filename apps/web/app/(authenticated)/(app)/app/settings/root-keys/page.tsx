import { PageHeader } from "@/components/dashboard/page-header";
import { RootKeyTable } from "@/components/dashboard/root-key-table";
import { getTenantId } from "@/lib/auth";
import { Key, db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";
import { CreateRootKeyButton } from "./create-root-key-button";

export const revalidate = 0;

export default async function SettingsKeysPage(props: {
  params: { apiId: string };
}) {
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

  const allKeys = await db.query.keys.findMany({
    where: eq(schema.keys.forWorkspaceId, workspace.id),
    limit: 100,
  });

  const keys: Key[] = [];
  const expired: Key[] = [];

  for (const k of allKeys) {
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
    <div className="min-h-screen ">
      <PageHeader
        title="Root Keys"
        description="Root keys are used to interact with the Unkey API."
        actions={[<CreateRootKeyButton key="create-root-key" apiId={props.params.apiId} />]}
      />
      <div className="grid grid-cols-1 gap-8 mb-20 w-full">
        <RootKeyTable data={keys} />
      </div>
    </div>
  );
}
