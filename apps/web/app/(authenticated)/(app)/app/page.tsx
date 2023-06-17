import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { redirect } from "next/navigation";

export default async function TenantOverviewPage() {
  const workspaceId = getTenantId();
  let workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.id, workspaceId),
  });
  if (!workspace) {
    redirect("/onboarding");
  }

  return (
    <div>
      <PageHeader title={workspace?.name ?? "N/A"} description="Your Workspace" />
    </div>
  );
}
