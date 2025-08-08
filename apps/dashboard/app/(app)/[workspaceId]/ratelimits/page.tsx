"use client";

import { RatelimitClient } from "./_components/ratelimit-client";
import { Navigation } from "./navigation";

export default async function RatelimitOverviewPage({ params }: { params: { workspaceId: string } }) {
  const { workspaceId } = params;
  return (
    <div>
      <Navigation workspaceId={workspaceId} />
      <RatelimitClient workspaceId={workspaceId} />
    </div>
  );
}
