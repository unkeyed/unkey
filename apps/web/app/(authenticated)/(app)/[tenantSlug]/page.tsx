import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound } from "next/navigation";

export default async function TenantOverviewPage() {
  const tenantId = getTenantId();
  const tenant = await db.query.tenants.findFirst({
    where: eq(schema.tenants.id, tenantId),
  });

  return (
    <div>
      <PageHeader title={tenant?.name ?? "N/A"} description="Your Team" />
    </div>
  );
}
