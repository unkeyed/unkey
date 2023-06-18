import { PageHeader } from "@/components/PageHeader";
import { CreateApiButton } from "./CreateAPI";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound } from "next/navigation";
import { Separator } from "@/components/ui/separator";
import { Row } from "./row";

export default async function TenantOverviewPage() {
  const workspaceId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.id, workspaceId),
    with: {
      apis: true,
    },
  });
  if (!workspace) {
    return notFound();
  }
  return (
    <div>
      <PageHeader
        title="Applications"
        description="Manage your different APIs"
        actions={[<CreateApiButton key="createApi" />]}
      />
      <Separator className="my-6" />

      {workspace.apis.map((api) => (
        <Row key={api.id} api={api} />
      ))}
    </div>
  );
}
