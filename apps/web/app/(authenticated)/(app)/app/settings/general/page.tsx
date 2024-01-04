import { CopyButton } from "@/components/dashboard/copy-button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { UpdateWorkspaceImage } from "./update-workspace-image";
import { UpdateWorkspaceName } from "./update-workspace-name";
export const dynamic = "force-dynamic";

export default async function SettingsPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div className="mb-20 flex flex-col gap-8 ">
      <UpdateWorkspaceName workspace={workspace} />
      <UpdateWorkspaceImage />
      <Card>
        <CardHeader>
          <CardTitle>Workspace ID</CardTitle>
          <CardDescription>This is your workspace id. It's used in some API calls.</CardDescription>
        </CardHeader>
        <CardContent>
          <Code className="flex h-8 w-full max-w-sm items-center justify-between gap-4">
            <pre>{workspace.id}</pre>
            <div className="flex items-start justify-between gap-4">
              <CopyButton value={workspace.id} />
            </div>
          </Code>
        </CardContent>
      </Card>
    </div>
  );
}
