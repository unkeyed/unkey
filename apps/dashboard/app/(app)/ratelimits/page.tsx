"use client";

import { trpc } from "@/lib/trpc/client";
import { RatelimitClient } from "./_components/ratelimit-client";
import { Navigation } from "./navigation";

export default function RatelimitOverviewPage() {
  const workspaceQuery = trpc.workspace.getWorkspace.useQuery();

  if (workspaceQuery.isLoading) {
    return <div>Loading workspace info...</div>;
  }

  if (workspaceQuery.error) {
    return <div>Error: {workspaceQuery.error.message}</div>;
  }

  const workspace = workspaceQuery.data;

  if (!workspace) {
    return <div>Workspace data not available.</div>;
  }

  return (
    <div>
      <Navigation />
      <RatelimitClient ratelimitNamespaces={(workspace as any).ratelimitNamespaces} />
    </div>
  );
}
