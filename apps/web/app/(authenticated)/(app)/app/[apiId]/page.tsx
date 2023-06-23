import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound, redirect } from "next/navigation";
import { DeleteApiButton } from "./DeleteApi";
import { Separator } from "@/components/ui/separator";
import Link from "next/link";
import { ApiKeyTable } from "@/components/ApiKeyTable";
import { Badge } from "@/components/ui/badge";
import { CopyButton } from "@/components/CopyButton";
import { Button } from "@/components/ui/button";
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
    return redirect("/onboarding");
  }

  return (
    <div>
      <PageHeader
        title={api.name}
        description={"Here is a list of your current API keys"}
        actions={[
          <Badge key="apiId" variant="outline" className="font-mono font-medium">
            {api.id}
            <CopyButton value={api.id} className="ml-2" />
          </Badge>,
          <Link href={`/app/${api.id}/keys/new`} ><Button>Create Key</Button></Link>,
          <DeleteApiButton key="delete-api" apiId={api.id} apiName={api.name} />,
        ]}
      />

      <Separator className="my-6" />

      <ApiKeyTable data={api.keys} />
    </div>
  );
}
