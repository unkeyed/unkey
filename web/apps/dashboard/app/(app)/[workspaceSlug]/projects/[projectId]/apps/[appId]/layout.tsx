"use client";
import type { PropsWithChildren } from "react";
import { ProjectDataProvider } from "./(overview)/data-provider";
import { PendingRedeployBanner } from "./components/pending-redeploy-banner";

export default function ProjectLayoutWrapper({ children }: PropsWithChildren) {
  return (
    <ProjectDataProvider>
      <div className="h-full flex flex-col overflow-hidden">
        <div className="flex flex-1 min-h-0">
          <div className="flex-1 overflow-auto">{children}</div>
        </div>
        <PendingRedeployBanner />
      </div>
    </ProjectDataProvider>
  );
}
