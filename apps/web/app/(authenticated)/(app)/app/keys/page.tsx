import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound, redirect } from "next/navigation";
import { CreateKeyButton } from "./CreateKey";
import { Separator } from "@/components/ui/separator";

import { ApiKeyTable } from "@/components/ApiKeyTable";
export default async function SettingsKeysPage() {
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
  });

  return (
    <div>
      <PageHeader
        title="Keys"
        description="These keys are used to interact with the unkey API"
        actions={[<CreateKeyButton key="create-key" apiId={workspace.apis.at(0)?.id} />]}
      />
      <Separator className="my-6" />

      <ApiKeyTable data={keys} />
    </div>
  );
}
