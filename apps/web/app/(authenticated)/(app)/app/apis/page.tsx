import { PageHeader } from "@/components/PageHeader";
import { CreateApiButton } from "./CreateAPI";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound } from "next/navigation";
import { Separator } from "@/components/ui/separator";
import { Row } from "./row";

export default async function TenantOverviewPage() {
  const tenantId = getTenantId();
  const tenant = await db.query.tenants.findFirst({
    where: eq(schema.tenants.id, tenantId),
    with: {
      apis: true,
    },
  });
  if (!tenant) {
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

      {tenant.apis.map((api) => (
        <Row key={api.id} api={api} />
      ))}
    </div>
  );
}
