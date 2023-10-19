import { PageHeader } from "@/components/dashboard/page-header";
import { RootKeyTable } from "@/components/dashboard/root-key-table";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";
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

  const keys = await db.query.keys.findMany({
    where: eq(schema.keys.forWorkspaceId, workspace.id),
    limit: 100,
  });

  return (
    <div className="min-h-screen">
      <PageHeader
        title="Root Keys"
        description="Root keys are used to interact with the Unkey API."
        actions={[<CreateRootKeyButton key="create-root-key" apiId={props.params.apiId} />]}
      />
      <RootKeyTable data={keys} />
    </div>
  );
}
