"use client";

import { DeploymentsNavigation } from "./navigation";

export default function ProjectDetails({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <div>
      <DeploymentsNavigation projectId={projectId} />
      <div className="flex flex-col">hoho</div>
    </div>
  );
}
