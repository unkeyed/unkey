"use client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { WorkspaceNavbar } from "../workspace-navbar";
import { CopyWorkspaceId } from "./copy-workspace-id";
import { UpdateWorkspaceName } from "./update-workspace-name";

/**
 * TODO: WorkOS doesn't have workspace images
 */

export default function SettingsPage() {
  const { workspace, isLoading } = useWorkspace();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !workspace) {
      router.replace("/new");
    }
  }, [isLoading, workspace, router]);

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-screen w-full">
        <Loading size={18} />
      </div>
    );
  }

  if (!workspace) {
    return null;
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
