import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";
export const runtime = "edge";
export default async function OnboardingPage() {
  const tenantId = getTenantId();
  const workspaces = await db.query.workspaces.findMany({
    where: eq(schema.workspaces.tenantId, tenantId),
  });

  if (workspaces.length > 0) {
    return redirect("/app/apis");
  }

  return redirect("/new");
}
