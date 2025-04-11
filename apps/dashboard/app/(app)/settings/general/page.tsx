import { PageContent } from "@/components/page-content";
import { Separator } from "@/components/ui/separator";
import { getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { WorkspaceNavbar } from "../workspace-navbar";
import { CopyWorkspaceId } from "./copy-workspace-id";
import { UpdateWorkspaceName } from "./update-workspace-name";

/**
 * TODO: WorkOS doesn't have workspace images
 */

export const dynamic = "force-dynamic";

export default async function SettingsPage() {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div>
      <WorkspaceNavbar workspace={workspace} activePage={{ href: "general", text: "General" }} />
      <PageContent>
        <div className="flex items-center justify-center w-full py-3 ">
          <div className="lg:w-[760px] flex-col justify-center items-center">
            <div className="w-full text-accent-12 font-semibold text-lg pt-[22px] pb-[20px] text-left border-b border-gray-4 px-2">
              Workspace Settings
            </div>
            <UpdateWorkspaceName workspace={workspace} />
            {/* <UpdateWorkspaceImage /> */}
            <Separator className="bg-gray-4" orientation="horizontal" />
            <CopyWorkspaceId workspaceId={workspace.id} />
          </div>
        </div>
      </PageContent>
    </div>
  );
}
