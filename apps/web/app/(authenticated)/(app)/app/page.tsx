import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { redirect } from "next/navigation";

export default async function TenantOverviewPage() {
  const tenantId = getTenantId();
  let tenant = await db.query.tenants.findFirst({
    where: eq(schema.tenants.id, tenantId),
  });
  if (!tenant) {
    redirect("/onboarding");
  }

  return (
    <div>
      <PageHeader title={tenant?.name ?? "N/A"} description="Your team" />
    </div>
  );
}
