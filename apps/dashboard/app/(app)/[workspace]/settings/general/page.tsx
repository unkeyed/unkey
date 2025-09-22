"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { WorkspaceNavbar } from "../workspace-navbar";
import { CopyWorkspaceId } from "./copy-workspace-id";
import { UpdateWorkspaceName } from "./update-workspace-name";

/**
 * TODO: WorkOS doesn't have workspace images
 */

export default function SettingsPage() {
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <WorkspaceNavbar activePage={{ href: "general", text: "General" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
          <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
            Workspace Settings
          </div>
          <div className="w-full flex flex-col">
            <UpdateWorkspaceName />
            {/* <UpdateWorkspaceImage /> */}
            {workspace && <CopyWorkspaceId workspaceId={workspace.id} />}
          </div>
        </div>
      </div>
    </div>
  );
}
