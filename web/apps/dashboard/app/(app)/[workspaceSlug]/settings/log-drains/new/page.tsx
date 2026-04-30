"use client";

import { LogDrainWizard } from "@/components/log-drains/log-drain-wizard";
import { useParams } from "next/navigation";
import { WorkspaceNavbar } from "../../workspace-navbar";

export default function NewWorkspaceLogDrainPage() {
  const params = useParams();
  const workspaceSlug = typeof params?.workspaceSlug === "string" ? params.workspaceSlug : null;

  if (!workspaceSlug) {
    return null;
  }

  return (
    <div>
      <WorkspaceNavbar activePage={{ href: "log-drains", text: "Log Drains" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col gap-5 mx-6 mt-4">
          <LogDrainWizard scope="workspace" workspaceSlug={workspaceSlug} />
        </div>
      </div>
    </div>
  );
}
