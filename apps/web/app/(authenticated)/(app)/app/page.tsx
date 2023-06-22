import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
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

  // return (
  //   <div>
  //     <PageHeader title={workspace.name} description="Your Workspace" />
  //   </div>
  // );
}
