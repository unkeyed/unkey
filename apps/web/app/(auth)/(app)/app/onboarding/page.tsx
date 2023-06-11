import { Onboarding } from "./client";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { redirect } from "next/navigation";

export default async function OnboardingPage() {
  const tenantId = getTenantId();
  let tenant = await db.query.tenants.findFirst({
    where: eq(schema.tenants.id, tenantId),
  });
  if (tenant) {
    redirect("/app");
  }

  return <Onboarding tenantId={tenantId} />;
}
