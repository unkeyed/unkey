import { CopyButton } from "@/components/dashboard/copy-button";
import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navigation } from "@/components/navigation/navigation";
import { PageContent } from "@/components/page-content";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Gear } from "@unkey/icons";
import { redirect } from "next/navigation";
import { navigation } from "../constants";
import { UpdateWorkspaceName } from "./update-workspace-name";
// import { UpdateWorkspaceImage } from "./update-workspace-image";

/**
 * TODO: WorkOS doesn't have workspace images
 */

export const dynamic = "force-dynamic";

export default async function SettingsPage() {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div>
      <Navigation href="/settings/general" name="Settings" icon={<Gear />} />
      <PageContent>
        <SubMenu navigation={navigation} segment="general" />
        <div className="mb-20 flex flex-col gap-8 mt-8">
          <UpdateWorkspaceName workspace={workspace} />
          {/* <UpdateWorkspaceImage /> */}
          <Card>
            <CardHeader>
              <CardTitle>Workspace ID</CardTitle>
              <CardDescription>
                This is your workspace id. It's used in some API calls.
              </CardDescription>
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
      </PageContent>
    </div>
  );
}
