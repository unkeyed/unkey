import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { redirect } from "next/navigation";
import { Onboarding } from "./client";
export default async function OnboardingPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  console.log("workspace", workspace);
  if (workspace) {
    return redirect("/app");
  }

  return <Onboarding tenantId={tenantId} />;
}
