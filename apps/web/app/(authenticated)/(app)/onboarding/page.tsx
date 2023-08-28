import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";

export default async function OnboardingPage() {
  const tenantId = getTenantId();
  const workspaces = await db.query.workspaces.findMany({
    where: eq(schema.workspaces.tenantId, tenantId),
  });

  if (workspaces.length > 0) {
    return redirect(`/${workspaces.at(0)!.slug}/apis`);
  }

  return redirect("/new");
}
