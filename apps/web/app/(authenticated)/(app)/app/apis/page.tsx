import { PageHeader } from "@/components/PageHeader";
import { CreateApiButton } from "./CreateAPI";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound } from "next/navigation";

export default async function TenantOverviewPage() {
  const tenantId = getTenantId();
  const tenant = await db.query.tenants.findFirst({
    where: eq(schema.tenants.id, tenantId),
  });
  if (!tenant) {
    return notFound();
  }
  return (
    <div>
      <PageHeader
        title="Applications"
        description="Manage your different APIs"
        actions={[<CreateApiButton key="createApi" tenant={{ slug: tenant.slug }} />]}
      />
    </div>
  );
}
