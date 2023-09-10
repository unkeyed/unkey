import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";

export default async function TenantOverviewPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/onboarding");
  }
  return redirect("/app/apis");
}
