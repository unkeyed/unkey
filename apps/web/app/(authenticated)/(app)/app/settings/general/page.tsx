import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";
import { UpdateWorkspaceName } from "./update-workspace-name";

export const revalidate = 0;

export default async function SettingsPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/onboarding");
  }

  return (
    <div className="flex flex-col gap-8">
      <UpdateWorkspaceName workspace={workspace} />
    </div>
  );
}
