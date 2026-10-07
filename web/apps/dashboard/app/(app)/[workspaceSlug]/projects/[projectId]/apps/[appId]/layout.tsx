"use client";
import type { PropsWithChildren } from "react";
import { ProjectDataProvider } from "./(overview)/data-provider";
import { PendingRedeployBanner } from "./components/pending-redeploy-banner";

export default function ProjectLayoutWrapper({ children }: PropsWithChildren) {
  return (
    <ProjectDataProvider>
      <div className="flex flex-1 flex-col">
        <div className="flex flex-1">
          <div className="min-w-0 flex-1">{children}</div>
        </div>
        <PendingRedeployBanner />
      </div>
    </ProjectDataProvider>
  );
}
