import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound } from "next/navigation";
import { currentUser } from "@clerk/nextjs";

export default async function TenantOverviewPage() {
  const tenantId = getTenantId();
  const user = await currentUser();
  let tenant = await db.query.tenants.findFirst({
    where: eq(schema.tenants.id, tenantId),
  });
  if (!tenant) {
    if (!user) {
      return notFound();
    }

    let slug = user.username!.replace(/[^a-zA-Z0-9_.-]/g, "-");

    // Replace consecutive dashes with a single dash
    slug = slug.replace(/-+/g, "-");

    // Remove leading and trailing dashes
    slug = slug.replace(/^-|-$/g, "");

    tenant = {
      id: tenantId,
      name: user.username!,
      slug,
    };
    await db.insert(schema.tenants).values(tenant);
  }

  return (
    <div>
      <PageHeader title={tenant?.name ?? "N/A"} description="Your team" />
    </div>
  );
}
