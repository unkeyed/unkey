"use client";
import { type PropsWithChildren, useState } from "react";
import { ProjectDataProvider } from "./(overview)/data-provider";
import { ProjectLayoutContext } from "./(overview)/layout-provider";
import { ProjectNavigation } from "./(overview)/navigations/project-navigation";
import { PendingRedeployBanner } from "./components/pending-redeploy-banner";

export default function ProjectLayoutWrapper({ children }: PropsWithChildren) {
  return <ProjectLayout>{children}</ProjectLayout>;
}

const ProjectLayout = ({ children }: PropsWithChildren) => {
  return (
    <ProjectDataProvider>
      <ProjectLayoutInner>{children}</ProjectLayoutInner>
    </ProjectDataProvider>
  );
};

const ProjectLayoutInner = ({ children }: PropsWithChildren) => {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  return (
    <ProjectLayoutContext.Provider
      value={{
        tableDistanceToTop,
      }}
    >
      <div className="h-full flex flex-col overflow-hidden">
        <ProjectNavigation onMount={setTableDistanceToTop} />
        <div className="flex flex-1 min-h-0">
          <div className="flex-1 overflow-auto">{children}</div>
        </div>
        <PendingRedeployBanner />
      </div>
    </ProjectLayoutContext.Provider>
  );
};
