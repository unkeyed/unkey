"use client";

import { LogDrainDetail } from "@/components/log-drains/log-drain-detail";
import { useParams } from "next/navigation";
import { WorkspaceNavbar } from "../../workspace-navbar";

export default function WorkspaceLogDrainDetailPage() {
  const params = useParams();
  const workspaceSlug = typeof params?.workspaceSlug === "string" ? params.workspaceSlug : null;
  const drainId = typeof params?.drainId === "string" ? params.drainId : null;

  if (!workspaceSlug || !drainId) {
    return null;
  }

  return (
    <div>
      <WorkspaceNavbar activePage={{ href: "log-drains", text: "Log Drains" }} />
      <div className="py-3 w-full flex items-center justify-center">
        <div className="w-[900px] flex flex-col gap-5 mx-6 mt-4">
          <LogDrainDetail scope="workspace" workspaceSlug={workspaceSlug} drainId={drainId} />
        </div>
      </div>
    </div>
  );
}
