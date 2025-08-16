"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { redirect, useRouter } from "next/navigation";
import { RatelimitClient } from "./_components/ratelimit-client";
import { Navigation } from "./navigation";

export default async function RatelimitOverviewPage({
  params,
}: { params: { workspaceId: string } }) {
  const { workspace } = useWorkspace();
  const router = useRouter();

  if (!workspace) {
    return redirect("/new");
  }

  if (workspace?.id !== params.workspaceId) {
    router.replace(`/${workspace.id}/ratelimits`);
  }

  return (
    <div>
      <Navigation workspaceId={workspace?.id ?? ""} />
      <RatelimitClient workspaceId={workspace?.id ?? ""} />
    </div>
  );
}
