import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound, redirect } from "next/navigation";
import { CreateKeyButton } from "./CreateKey";
import { Separator } from "@/components/ui/separator";

import { ApiKeyTable } from "@/components/ApiKeyTable";
export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
      keys: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return notFound();
  }

  return (
    <div>
      <PageHeader
        title="Keys"
        description="All API keys belonging to this api"
        actions={[<CreateKeyButton key="create-key" apiId={props.params.apiId} />]}
      />
      <Separator className="my-6" />

      <ApiKeyTable data={api.keys} />
    </div>
  );
}
