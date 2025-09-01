import { getAuthOrRedirect } from "@/lib/auth";
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
  const { orgId } = await getAuthOrRedirect();
  if (!orgId) {
    redirect("/new");
  }
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });
  if (!workspace) {
    redirect("/new");
  }

  return (
    <div>
      <WorkspaceNavbar workspace={workspace} activePage={{ href: "general", text: "General" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
          <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
            Workspace Settings
          </div>
          <div className="w-full flex flex-col">
            <UpdateWorkspaceName workspace={workspace} />
            {/* <UpdateWorkspaceImage /> */}
            <CopyWorkspaceId workspaceId={workspace.id} />
          </div>
        </div>
      </div>
    </div>
  );
}
